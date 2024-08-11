package agent

import (
	"flag"
)

var flagServerAddress string
var flagPollInterval int
var flagReportInterval int

func parseFlags() {
	flag.StringVar(&flagServerAddress, "a", "localhost:8080", "address and port to send reports")
	flag.IntVar(&flagPollInterval, "p", 2, "poll interval in seconds")
	flag.IntVar(&flagReportInterval, "r", 10, "report interval in seconds")
	flag.Parse()
}
