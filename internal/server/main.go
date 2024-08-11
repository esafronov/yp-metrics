package server

import (
	"net/http"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/storage"
)

func Run() error {
	parseFlags()

	h := handlers.NewAPIHandler(storage.NewMemStorage())

	var srv = http.Server{
		Addr:    flagRunAddress,
		Handler: h.GetRouter(),
	}

	return srv.ListenAndServe()

}
