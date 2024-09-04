package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/storage"
)

var serverAddress string   //server address to listen
var storeInterval *int     //store interval
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
			fmt.Printf("storage can't be closed %s", err)
		}
	}()
	h := handlers.NewAPIHandler(storage)
	srv := http.Server{
		Addr:    serverAddress,
		Handler: h.GetRouter(),
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
		s := <-sigs
		fmt.Println("got signal ", s)
		srv.Close()
	}()
	fmt.Printf("listen on address: %s\r\n", serverAddress)
	fmt.Println("file storage:", fileStoragePath)
	fmt.Println("storage interval:", *storeInterval)
	fmt.Println("restore flag:", *restoreData)
	return srv.ListenAndServe()
}
