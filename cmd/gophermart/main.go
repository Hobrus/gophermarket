package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

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

	pool, err := pgxpool.New(ctx, cfg.DatabaseURI)
	if err != nil {
		log.Fatal(err)
	}

	userRepo, orderRepo, withdrawalRepo := postgres.New(pool)

	authSvc := service.NewAuthService(userRepo, []byte(cfg.JWTSecret))
	orderSvc := service.NewOrderService(orderRepo)
	balanceSvc := service.NewBalanceService(orderRepo, withdrawalRepo)
	withdrawSvc := service.NewWithdrawService(orderRepo, withdrawalRepo)
	updater := service.NewOrderUpdater(orderRepo, accrualclient.New(cfg.AccrualAddress))

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.Middleware(l))
	router.Use(middleware.Gzip(5))

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
