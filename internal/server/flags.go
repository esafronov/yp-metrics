package server

import (
	"flag"
)

func parseFlags() {
	if serverAddress == "" {
		flag.StringVar(&serverAddress, "a", "localhost:8080", "address and port to run server")
	}
	if storeInterval == nil {
		storeInterval = flag.Int("i", 300, "interval for backuping data")
	}
	if fileStoragePath == "" {
		flag.StringVar(&fileStoragePath, "f", "", "filepath to store backup data")
	}
	if restoreData == nil {
		restoreData = flag.Bool("r", true, "restore data on server start")
	}
	if databaseDsn == nil {
		databaseDsn = flag.String("d", "", "database dsn")
	}
	flag.Parse()
}
