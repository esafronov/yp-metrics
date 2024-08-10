package server

import (
	"net/http"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/storage"
)

func Run() {
	h := handlers.NewAPIHandler(storage.NewMemStorage())

	var srv = http.Server{
		Addr:    ":8080",
		Handler: h.GetRouter(),
	}

	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
