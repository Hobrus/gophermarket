package accrualclient

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"github.com/shopspring/decimal"
	"golang.org/x/time/rate"
)

// Client requests order accrual information from the external service.
type Client interface {
	// Get retrieves accrual status for order number. If the service responds
	// with 429 Too Many Requests it returns a TooManyRequestsError containing
	// the suggested retry delay.
	Get(ctx context.Context, number string) (status string, accrual *decimal.Decimal, err error)
}

// TooManyRequestsError is returned when the external service responds with
// HTTP 429. It contains the duration to wait before sending the next request.
type TooManyRequestsError struct {
	RetryAfter time.Duration
}

func (e TooManyRequestsError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("too many requests, retry after %v", e.RetryAfter)
	}
	return "too many requests"
}

// HTTPClient implements Client using net/http.
type HTTPClient struct {
	baseURL string
	http    *http.Client
	limiter *rate.Limiter
}

// New creates a new HTTPClient with provided base URL.
func New(baseURL string) *HTTPClient {
	lim := rate.NewLimiter(rate.Limit(5), 5)
	lim.AllowN(time.Now(), 5)
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport, otelhttp.WithTracerProvider(otel.GetTracerProvider())),
		},
		limiter: lim,
	}
}

type getResponse struct {
	Order   string           `json:"order"`
	Status  string           `json:"status"`
	Accrual *decimal.Decimal `json:"accrual"`
}

// Get implements Client using GET /api/orders/{number} request.
func (c *HTTPClient) Get(ctx context.Context, number string) (string, *decimal.Decimal, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return "", nil, err
	}
	url := fmt.Sprintf("%s/api/orders/%s", c.baseURL, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return "", nil, nil
	case http.StatusTooManyRequests:
		raStr := resp.Header.Get("Retry-After")
		var dur time.Duration
		if sec, err := strconv.Atoi(raStr); err == nil {
			dur = time.Duration(sec) * time.Second
		}
		return "", nil, TooManyRequestsError{RetryAfter: dur}
	}

	if resp.StatusCode > 299 {
		return "", nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var body io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", nil, err
		}
		defer gz.Close()
		body = gz
	}

	var gr getResponse
	if err := json.NewDecoder(body).Decode(&gr); err != nil {
		return "", nil, err
	}
	return gr.Status, gr.Accrual, nil
}

var _ Client = (*HTTPClient)(nil)
