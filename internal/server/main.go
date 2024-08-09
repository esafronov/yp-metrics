package server

import (
	"net/http"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/storage"
)

func Run() {
	var h = handlers.NewApiHandler(storage.NewMemStorage())
	m := http.NewServeMux()
	m.HandleFunc("/update/", h.Update)

	var srv = &http.Server{
		Addr:    ":8080",
		Handler: m,
	}

	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
