package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Hobrus/gophermarket/internal/accrualclient"
	dhttp "github.com/Hobrus/gophermarket/internal/delivery/http"
	"github.com/Hobrus/gophermarket/internal/service"
	"github.com/Hobrus/gophermarket/internal/storage/postgres"
	"github.com/Hobrus/gophermarket/pkg/logger"
	"github.com/Hobrus/gophermarket/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

var (
	ctx      context.Context
	cancel   context.CancelFunc
	pgC      testcontainers.Container
	accrualC testcontainers.Container
	pool     *pgxpool.Pool
	srv      *http.Server
	baseURL  string
	client   *http.Client
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	pgReq := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		Env:          map[string]string{"POSTGRES_PASSWORD": "pass", "POSTGRES_USER": "user", "POSTGRES_DB": "test"},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	}
	var err error
	pgC, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: pgReq, Started: true})
	Expect(err).NotTo(HaveOccurred())

	host, err := pgC.Host(ctx)
	Expect(err).NotTo(HaveOccurred())
	port, err := pgC.MappedPort(ctx, "5432/tcp")
	Expect(err).NotTo(HaveOccurred())
	dsn := fmt.Sprintf("postgres://user:pass@%s:%s/test?sslmode=disable", host, port.Port())
	pool, err = pgxpool.New(ctx, dsn)
	Expect(err).NotTo(HaveOccurred())
	ctxPing, cancelPing := context.WithTimeout(ctx, 10*time.Second)
	defer cancelPing()
	Expect(pool.Ping(ctxPing)).To(Succeed())

	b, err := os.ReadFile(filepath.Join("migrations", "0001_init.up.sql"))
	Expect(err).NotTo(HaveOccurred())
	_, err = pool.Exec(ctx, string(b))
	Expect(err).NotTo(HaveOccurred())

	script, err := os.ReadFile(filepath.Join("e2e", "accrual_server.js"))
	Expect(err).NotTo(HaveOccurred())
	accrualReq := testcontainers.ContainerRequest{
		Image:        "node:lts-alpine",
		ExposedPorts: []string{"3000/tcp"},
		Files: []testcontainers.ContainerFile{{
			Reader:            strings.NewReader(string(script)),
			ContainerFilePath: "/server.js",
			FileMode:          0644,
		}},
		Cmd:        []string{"node", "/server.js"},
		WaitingFor: wait.ForListeningPort("3000/tcp"),
	}
	accrualC, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: accrualReq, Started: true})
	Expect(err).NotTo(HaveOccurred())
	aHost, err := accrualC.Host(ctx)
	Expect(err).NotTo(HaveOccurred())
	aPort, err := accrualC.MappedPort(ctx, "3000/tcp")
	Expect(err).NotTo(HaveOccurred())
	accrualAddr := fmt.Sprintf("http://%s:%s", aHost, aPort.Port())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	baseURL = "http://" + ln.Addr().String()

	userRepo, orderRepo, withdrawalRepo := postgres.New(pool)
	authSvc := service.NewAuthService(userRepo, []byte("secret"))
	orderSvc := service.NewOrderService(orderRepo)
	balanceSvc := service.NewBalanceService(orderRepo, withdrawalRepo)
	withdrawSvc := service.NewWithdrawService(orderRepo, withdrawalRepo, balanceSvc)
	updater := service.NewOrderUpdater(orderRepo, accrualclient.New(accrualAddr), balanceSvc)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	l := logger.Init("info")
	router.Use(logger.Middleware(l))
	router.Use(middleware.Gzip(5))
	router.Mount("/", dhttp.NewRouter(authSvc))
	router.Group(func(r chi.Router) {
		r.Use(dhttp.JWT([]byte("secret")))
		r.Mount("/", dhttp.NewOrderRouter(orderSvc))
		r.Get("/api/user/balance", dhttp.Balance(balanceSvc))
		r.Post("/api/user/balance/withdraw", dhttp.Withdraw(withdrawSvc))
	})

	srv = &http.Server{Handler: router}
	go updater.Run(ctx, 1, 5, time.Second)
	go srv.Serve(ln)

	jar, _ := cookiejar.New(nil)
	client = &http.Client{Jar: jar}
})

var _ = AfterSuite(func() {
	cancel()
	if srv != nil {
		srv.Shutdown(context.Background())
	}
	if pool != nil {
		pool.Close()
	}
	if accrualC != nil {
		accrualC.Terminate(context.Background())
	}
	if pgC != nil {
		pgC.Terminate(context.Background())
	}
})

var _ = Describe("E2E", func() {
	It("processes full scenario", func() {
		resp, err := client.Post(baseURL+"/api/user/register", "application/json", strings.NewReader(`{"login":"u","password":"p"}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		resp.Body.Close()

		req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/user/orders", strings.NewReader("79927398713"))
		for _, c := range client.Jar.Cookies(req.URL) {
			req.AddCookie(c)
		}
		resp, err = client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
		resp.Body.Close()

		Eventually(func() float64 {
			req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/user/balance", nil)
			for _, c := range client.Jar.Cookies(req.URL) {
				req.AddCookie(c)
			}
			r, err := client.Do(req)
			if err != nil {
				return 0
			}
			defer r.Body.Close()
			if r.StatusCode != http.StatusOK {
				return 0
			}
			var b struct {
				Current float64 `json:"current"`
			}
			if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
				return 0
			}
			return b.Current
		}, 5*time.Second, 500*time.Millisecond).Should(Equal(1000.0))

		body := `{"order":"12345678903","sum":600}`
		req, _ = http.NewRequest(http.MethodPost, baseURL+"/api/user/balance/withdraw", strings.NewReader(body))
		for _, c := range client.Jar.Cookies(req.URL) {
			req.AddCookie(c)
		}
		resp, err = client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		req, _ = http.NewRequest(http.MethodGet, baseURL+"/api/user/balance", nil)
		for _, c := range client.Jar.Cookies(req.URL) {
			req.AddCookie(c)
		}
		resp, err = client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		var bal struct {
			Current   float64 `json:"current"`
			Withdrawn float64 `json:"withdrawn"`
		}
		json.NewDecoder(resp.Body).Decode(&bal)
		resp.Body.Close()

		Expect(bal.Current).To(Equal(400.0))
		Expect(bal.Withdrawn).To(Equal(600.0))
	})
})
