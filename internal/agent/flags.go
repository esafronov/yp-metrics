package agent

import (
	"flag"
)

func parseFlags() {
	serverAddressFlag := flag.String("a", "localhost:8080", "address and port to send reports")
	if serverAddress == nil {
		serverAddress = serverAddressFlag
	}
	pollIntervalFlag := flag.Int("p", 2, "poll interval in seconds")
	if pollInterval == nil {
		pollInterval = pollIntervalFlag
	}
	reportIntervalFlag := flag.Int("r", 10, "report interval in seconds")
	if reportInterval == nil {
		reportInterval = reportIntervalFlag
	}
	flag.Parse()
}
