package main

import (
	"github.com/esafronov/yp-metrics/internal/agent"
)

func main() {
	const (
		pollInterval   int = 2
		reportInterval int = 10
	)
	agent.Run(pollInterval, reportInterval)
}
