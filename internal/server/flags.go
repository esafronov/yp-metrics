package server

import (
	"flag"
)

func parseFlags() {

	serverAddressFlag := flag.String("a", "localhost:8080", "address and port to run server")
	if serverAddress == nil {
		serverAddress = serverAddressFlag
	}

	storeIntervalFlag := flag.Int("i", 300, "interval for backuping data")
	if storeInterval == nil {
		storeInterval = storeIntervalFlag
	}

	fileStoragePathFlag := flag.String("f", "", "filepath to store backup data")
	if fileStoragePath == nil {
		fileStoragePath = fileStoragePathFlag
	}

	restoreDataFlag := flag.Bool("r", true, "restore data on server start")
	if restoreData == nil {
		restoreData = restoreDataFlag
	}

	databaseDsnFlag := flag.String("d", "", "database dsn")
	if databaseDsn == nil {
		databaseDsn = databaseDsnFlag
	}

	secretKeyFlag := flag.String("k", "", "secret key for signature check")
	if secretKey == nil {
		secretKey = secretKeyFlag
	}

	profileServerAddressFlag := flag.String("ad", "", "profile server address to listen")
	if profileServerAddress == nil {
		profileServerAddress = profileServerAddressFlag
	}

	flag.Parse()
}
