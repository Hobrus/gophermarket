package main

// @title Gophermart API
// @version 1.0
// @BasePath /

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riandyrn/otelchi"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.1"

	_ "github.com/Hobrus/gophermarket/docs"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/Hobrus/gophermarket/internal/accrualclient"
	"github.com/Hobrus/gophermarket/internal/config"
	dhttp "github.com/Hobrus/gophermarket/internal/delivery/http"
	"github.com/Hobrus/gophermarket/internal/service"
	"github.com/Hobrus/gophermarket/internal/storage/postgres"
	"github.com/Hobrus/gophermarket/pkg/logger"
	"github.com/Hobrus/gophermarket/pkg/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	l := logger.Init("info")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	traceExp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	metricExp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpoint(endpoint), otlpmetrichttp.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	res, err := resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(semconv.ServiceName("gophermart")),
	)
	if err != nil {
		log.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	tr := tp.Tracer("gophermart")
	_ = tr

	defer func() {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
	}()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURI)
	if err != nil {
		log.Fatal(err)
	}
	poolCfg.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerProvider(tp),
		otelpgx.WithMeterProvider(mp),
	)
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatal(err)
	}
	if err := otelpgx.RecordStats(pool, otelpgx.WithStatsMeterProvider(mp)); err != nil {
		log.Fatal(err)
	}

	userRepo, orderRepo, withdrawalRepo := postgres.New(pool)

	authSvc := service.NewAuthService(userRepo, []byte(cfg.JWTSecret))
	orderSvc := service.NewOrderService(orderRepo)
	balanceSvc := service.NewBalanceService(orderRepo, withdrawalRepo)
	withdrawSvc := service.NewWithdrawService(orderRepo, withdrawalRepo, balanceSvc)
	updater := service.NewOrderUpdater(orderRepo, accrualclient.New(cfg.AccrualAddress), balanceSvc)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.Middleware(l))
	router.Use(middleware.Gzip(5))
	router.Use(otelchi.Middleware("gophermart", otelchi.WithTracerProvider(tp)))

	router.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	router.Mount("/health", dhttp.NewHealthRouter(pool))

	router.Mount("/", dhttp.NewRouter(authSvc))
	router.Group(func(r chi.Router) {
		r.Use(dhttp.JWT([]byte(cfg.JWTSecret)))
		r.Mount("/", dhttp.NewOrderRouter(orderSvc))
		r.Mount("/", dhttp.NewOrdersRouter(orderRepo))
		r.Get("/api/user/balance", dhttp.Balance(balanceSvc))
		r.Post("/api/user/balance/withdraw", dhttp.Withdraw(withdrawSvc))
		r.Get("/api/user/withdrawals", dhttp.Withdrawals(withdrawalRepo))
	})

	go updater.Run(ctx, 2, 5, time.Second)
	go http.ListenAndServe(cfg.RunAddress, router)

	<-ctx.Done()
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool.Close()
	<-ctxShutdown.Done()
}
