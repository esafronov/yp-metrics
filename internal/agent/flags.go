package agent

import (
	"flag"
)

func parseFlags() {
	if serverAddress == "" {
		flag.StringVar(&serverAddress, "a", "localhost:8080", "address and port to send reports")
	}
	if pollInterval != -1 {
		flag.IntVar(&pollInterval, "p", 2, "poll interval in seconds")
	}
	if reportInterval != -1 {
		flag.IntVar(&reportInterval, "r", 10, "report interval in seconds")
	}
	flag.Parse()
}
