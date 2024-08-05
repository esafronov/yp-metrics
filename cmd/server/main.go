package main

import (
	"net/http"

	"github.com/esafronov/yp-metrics/internal/handlers"
)

func main() {
	var h handlers.Main
	err := http.ListenAndServe(`:8080`, h)
	if err != nil {
		panic(err)
	}
}
