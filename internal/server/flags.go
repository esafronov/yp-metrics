package server

import (
	"flag"
)

var _restoreData bool
var _storeInterval int

func parseFlags() {
	if serverAddress == "" {
		flag.StringVar(&serverAddress, "a", "localhost:8080", "address and port to run server")
	}
	if storeInterval == nil {
		storeInterval = &_storeInterval
		flag.IntVar(storeInterval, "i", 300, "interval for backuping data")
	}
	if fileStoragePath == "" {
		flag.StringVar(&fileStoragePath, "f", "", "filepath to store backup data")
	}
	if restoreData == nil {
		restoreData = &_restoreData
		flag.BoolVar(restoreData, "r", true, "restore data on server start")
	}
	flag.Parse()
}
