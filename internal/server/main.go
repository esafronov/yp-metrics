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
	"github.com/esafronov/yp-metrics/internal/server/config"
	"github.com/esafronov/yp-metrics/internal/storage"
	"go.uber.org/zap"
)

func Run() error {
	config.Initialize()
	params := config.Params
	logger.Log.Info("params",
		zap.String("Address", *params.Address),
		zap.String("DatabaseDsn", *params.DatabaseDsn),
		zap.String("FileStoragePath", *params.FileStoragePath),
		zap.String("ProfileServerAddress", *params.ProfileServerAddress),
		zap.Bool("Restore", *params.Restore),
		zap.Int("StoreInterval", *params.StoreInterval),
		zap.String("SecretKey", *params.SecretKey),
		zap.String("CryptoKey", *params.CryptoKey),
	)
	err := pg.Connect(params.DatabaseDsn)
	if err != nil {
		return err
	}
	ctx := context.Background()
	var storageInst storage.Repositories
	if params.DatabaseDsn != nil && *params.DatabaseDsn == "" {
		storageInst, err = storage.NewHybridStorage(ctx, params.FileStoragePath, params.StoreInterval, params.Restore)
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
	if params.ProfileServerAddress != nil && *params.ProfileServerAddress != "" {
		profileServer := pprofserv.NewDebugServer(*params.ProfileServerAddress)
		profileServer.Start()
		defer profileServer.Close()
	}
	h := handlers.NewAPIHandler(
		storageInst,
		handlers.OptionWithSecretKey(*params.SecretKey),
		handlers.OptionWithCryptoKey(*params.CryptoKey),
	)
	if params.Address == nil {
		return errors.New("serverAddress is nil")
	}
	srv := http.Server{
		Addr:    *params.Address,
		Handler: h.GetRouter(),
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGQUIT)
	idleConnsClosed := make(chan struct{})
	go func() {
		s := <-sigs
		fmt.Println("got signal ", s)
		err := srv.Shutdown(context.Background())
		if err != nil {
			logger.Log.Info(err.Error())
		}
		close(idleConnsClosed)
	}()
	err = srv.ListenAndServe()
	<-idleConnsClosed
	return err
}
