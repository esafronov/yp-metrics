package server

import (
	"fmt"
	"net/http"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/storage"
)

var serverAddress string   //server address to listen
var storeInterval int = -1 //store interval
var fileStoragePath string //file storage path
var restoreData *bool      //restore or not data on start

func Run() error {
	if err := parseEnv(); err != nil {
		return err
	}
	parseFlags()
	err := logger.Initialize("debug")
	if err != nil {
		return err
	}
	storage, err := storage.NewHybridStorage(fileStoragePath, storeInterval, restoreData)
	if err != nil {
		return err
	}
	defer func() {
		err := storage.Close()
		if err != nil {
			fmt.Printf("storage can't be closed %v", err)
		}
	}()
	h := handlers.NewAPIHandler(storage)
	srv := http.Server{
		Addr:    serverAddress,
		Handler: h.GetRouter(),
	}
	fmt.Printf("listen on address: %s", serverAddress)
	return srv.ListenAndServe()
}
