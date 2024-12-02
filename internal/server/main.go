// Server package process incomming requests from Agent
package server

import (
	"context"
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
	if *databaseDsn == "" {
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
	if *profileServerAddress != "" {
		profileServer := pprofserv.NewDebugServer(*profileServerAddress)
		profileServer.Start()
		defer profileServer.Close()
	}

	h := handlers.NewAPIHandler(storageInst, *secretKey)
	srv := http.Server{
		Addr:    *serverAddress,
		Handler: h.GetRouter(),
	}
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
		s := <-sigs
		fmt.Println("got signal ", s)
		srv.Close()
	}()
	fmt.Printf("listen on address: %s\r\n", *serverAddress)
	fmt.Println("file storage:", *fileStoragePath)
	fmt.Println("storage interval:", *storeInterval)
	fmt.Println("restore flag:", *restoreData)
	fmt.Println("database dsn:", *databaseDsn)
	fmt.Println("key:", *secretKey)
	return srv.ListenAndServe()
}
