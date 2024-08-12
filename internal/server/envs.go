package server

import (
	"github.com/caarlos0/env/v6"
)

type envParams struct {
	Address string `env:"ADDRESS"`
}

func parseEnv() error {
	var p envParams
	if err := env.Parse(&p); err != nil {
		return err
	}
	serverAddress = p.Address
	return nil
}
