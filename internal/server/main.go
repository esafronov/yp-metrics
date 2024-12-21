// Package server process incomming requests from Agent
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/pg"
	"github.com/esafronov/yp-metrics/internal/pprofserv"
	"github.com/esafronov/yp-metrics/internal/storage"
)

var serverAddress *string        //server address to listen
var storeInterval *int           //store interval
var fileStoragePath *string      //file storage path
var restoreData *bool            //restore or not data on start
var databaseDsn *string          //db connection dsn
var secretKey *string            //secret key for signature check
var profileServerAddress *string //profile serveraddress to listen

func Run() error {
	if err := parseEnv(); err != nil {
		return err
	}
	parseFlags()
	err := logger.Initialize("debug")
	if err != nil {
		return err
	}
	err = pg.Connect(databaseDsn)
	if err != nil {
		return err
	}
	ctx := context.Background()
	var storageInst storage.Repositories
	if databaseDsn != nil && *databaseDsn == "" {
		storageInst, err = storage.NewHybridStorage(ctx, fileStoragePath, storeInterval, restoreData)
		if err != nil {
			return err
		}
	} else {
		storageInst, err = storage.NewDBStorage(ctx, pg.DB)
		if err != nil {
			return err
		}
	}
	defer func() {
		err := storageInst.Close(ctx)
		if err != nil {
			fmt.Printf("storage can't be closed %s", err)
		}
	}()

	//run profile server if env/flag is set
	if profileServerAddress != nil && *profileServerAddress != "" {
		profileServer := pprofserv.NewDebugServer(*profileServerAddress)
		profileServer.Start()
		defer profileServer.Close()
	}
	if secretKey == nil {
		panic("secretKey is nil")
	}
	h := handlers.NewAPIHandler(storageInst, *secretKey)
	if serverAddress == nil {
		return errors.New("serverAddress is nil")
	}
	srv := http.Server{
		Addr:    *serverAddress,
		Handler: h.GetRouter(),
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
		s := <-sigs
		fmt.Println("got signal ", s)
		err := srv.Close()
		if err != nil {
			logger.Log.Info(err.Error())
		}
	}()
	if serverAddress != nil {
		fmt.Printf("listen on address: %s\r\n", *serverAddress)
	}
	if fileStoragePath != nil {
		fmt.Println("file storage:", *fileStoragePath)
	}
	if storeInterval != nil {
		fmt.Println("storage interval:", *storeInterval)
	}
	if restoreData != nil {
		fmt.Println("restore flag:", *restoreData)
	}
	if databaseDsn != nil {
		fmt.Println("database dsn:", *databaseDsn)
	}
	if secretKey != nil {
		fmt.Println("key:", *secretKey)
	}
	return srv.ListenAndServe()
}
