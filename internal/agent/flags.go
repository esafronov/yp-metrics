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
	secretKeyFlag := flag.String("k", "", "secret key for request signing")
	if secretKey == nil {
		secretKey = secretKeyFlag
	}
	rateLimitFlag := flag.Int("l", 0, "max parallel request limit, 0 = send in batch")
	if rateLimit == nil {
		rateLimit = rateLimitFlag
	}
	profileServerAddressFlag := flag.String("ad", "", "profile server address to listen")
	if profileServerAddress == nil {
		profileServerAddress = profileServerAddressFlag
	}
	flag.Parse()
}
