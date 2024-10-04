package agent

import (
	"github.com/caarlos0/env/v6"
)

type envParams struct {
	Address        *string `env:"ADDRESS"`
	ReportInterval *int    `env:"REPORT_INTERVAL"`
	PollInterval   *int    `env:"POLL_INTERVAL"`
	SecretKey      *string `env:"KEY"`
	RateLimit      *int    `env:"RATE_LIMIT"`
}

func parseEnv() error {
	var p envParams
	if err := env.Parse(&p); err != nil {
		return err
	}
	serverAddress = p.Address
	reportInterval = p.ReportInterval
	pollInterval = p.PollInterval
	secretKey = p.SecretKey
	rateLimit = p.RateLimit
	return nil
}
