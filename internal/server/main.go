package server

import (
	"net/http"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/storage"
)

func Run() {
	var h = handlers.Main{
		Storage: storage.NewMemStorage(),
	}
	err := http.ListenAndServe(`:8080`, h)
	if err != nil {
		panic(err)
	}
}
