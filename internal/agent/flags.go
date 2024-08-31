package agent

import (
	"flag"
)

func parseFlags() {
	if serverAddress == "" {
		flag.StringVar(&serverAddress, "a", "localhost:8080", "address and port to send reports")
	}
	if pollInterval == nil {
		pollInterval = flag.Int("p", 2, "poll interval in seconds")
	}
	if reportInterval == nil {
		reportInterval = flag.Int("r", 10, "report interval in seconds")
	}
	flag.Parse()
}
