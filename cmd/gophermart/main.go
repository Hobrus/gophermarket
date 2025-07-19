package main

import (
	"log"
	"net/http"

	"github.com/Hobrus/gophermarket/pkg/logger"
	"github.com/Hobrus/gophermarket/pkg/middleware"
)

func main() {
	l := logger.Init("info")

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequestID(logger.Middleware(l)(middleware.Logging(mux)))

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
