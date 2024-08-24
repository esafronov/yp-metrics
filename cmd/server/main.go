package main

import (
	"log"

	"github.com/esafronov/yp-metrics/internal/server"
)

func main() {
	log.Fatal(server.Run())
}
