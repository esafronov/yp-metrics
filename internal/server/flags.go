package server

import (
	"flag"
)

func parseFlags() {
	if serverAddress != "" {
		return
	}
	flag.StringVar(&serverAddress, "a", "localhost:8080", "address and port to run server")
	flag.Parse()
}
