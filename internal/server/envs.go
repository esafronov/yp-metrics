package server

import (
	"github.com/caarlos0/env/v6"
)

type envParams struct {
	Address         string  `env:"ADDRESS"`
	StoreInterval   *int    `env:"STORE_INTERVAL"`
	FileStoragePath string  `env:"FILE_STORAGE_PATH"`
	Restore         *bool   `env:"RESTORE"`
	DatabaseDsn     *string `env:"DATABASE_DSN"`
}

func parseEnv() error {
	var p envParams
	if err := env.Parse(&p); err != nil {
		return err
	}
	serverAddress = p.Address
	storeInterval = p.StoreInterval
	fileStoragePath = p.FileStoragePath
	restoreData = p.Restore
	databaseDsn = p.DatabaseDsn
	return nil
}
