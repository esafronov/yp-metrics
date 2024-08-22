package server

import (
	"fmt"
	"net/http"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/storage"
)

var serverAddress string

func Run() error {
	if err := parseEnv(); err != nil {
		return err
	}
	parseFlags()
	err := logger.Initialize("debug")
	if err != nil {
		return err
	}
	h := handlers.NewAPIHandler(storage.NewMemStorage())
	srv := http.Server{
		Addr:    serverAddress,
		Handler: h.GetRouter(),
	}
	fmt.Printf("listen on address: %s", serverAddress)
	return srv.ListenAndServe()
}
