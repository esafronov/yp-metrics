// Package server process incomming requests from Agent
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/esafronov/yp-metrics/internal/access"
	pb "github.com/esafronov/yp-metrics/internal/grpc/proto"
	srv "github.com/esafronov/yp-metrics/internal/grpc/server"
	"github.com/esafronov/yp-metrics/internal/handlers"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/pg"
	"github.com/esafronov/yp-metrics/internal/pprofserv"
	"github.com/esafronov/yp-metrics/internal/server/config"
	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"google.golang.org/grpc/credentials"
	// Installing the gzip encoding registers it as an available compressor.
	// gRPC will automatically negotiate and use gzip if the client supports it.
	_ "google.golang.org/grpc/encoding/gzip"
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
		zap.String("CryptoCert", *params.CryptoCert),
		zap.String("TrustedSubnet", *params.TrustedSubnet),
		zap.Bool("UseGRPC", *params.UseGRPC),
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
	if *params.UseGRPC {
		err = runGRPCServer(params, storageInst)
	} else {
		err = runHTTPServer(params, storageInst)
	}
	return err
}

func runGRPCServer(params *config.AppParams, storageInst storage.Repositories) error {
	if params.Address == nil {
		return errors.New("serverAddress is nil")
	}

	// определяем порт для сервера
	listen, err := net.Listen("tcp", *params.Address)
	if err != nil {
		return err
	}

	var creds credentials.TransportCredentials
	if *params.CryptoKey != "" && *params.CryptoCert != "" {
		// Create tls based credential.
		creds, err = credentials.NewServerTLSFromFile(*params.CryptoCert, *params.CryptoKey)
		if err != nil {
			return err
		}
	}

	// создаём gRPC-сервер без зарегистрированной службы
	server := grpc.NewServer(
		//устанавливаем credentials
		grpc.Creds(creds),
		//цепочку интерсептеров
		grpc.ChainUnaryInterceptor(
			logger.UnaryLoggerInterceptor,
			access.UnaryValidateIpInterceptor(*params.TrustedSubnet),
			signing.UnaryValidateSignatureInterceptor(*params.SecretKey),
		),
	)

	//создаем метрик сервис
	service := srv.NewMetricsServer(
		storageInst,
		srv.OptionWithSecretKey(*params.SecretKey),
		srv.OptionWithCryptoKey(*params.CryptoKey),
		srv.OptionWithTrustedSubnet(*params.TrustedSubnet),
	)

	// регистрируем сервис на сервере
	pb.RegisterMetricsServer(server, service)

	fmt.Println("Сервер gRPC начал работу")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGQUIT)
	idleConnsClosed := make(chan struct{})
	go func() {
		s := <-sigs
		fmt.Println("got signal ", s)
		server.GracefulStop()
		fmt.Println("Сервер gRPC закончил работу")
		close(idleConnsClosed)
	}()
	// получаем запрос gRPC
	err = server.Serve(listen)
	<-idleConnsClosed
	return err
}

func runHTTPServer(params *config.AppParams, storageInst storage.Repositories) error {
	h := handlers.NewAPIHandler(
		storageInst,
		handlers.OptionWithSecretKey(*params.SecretKey),
		handlers.OptionWithCryptoKey(*params.CryptoKey),
		handlers.OptionWithTrustedSubnet(*params.TrustedSubnet),
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
		fmt.Println("HTTP сервер закончил работу")
		close(idleConnsClosed)
	}()
	fmt.Println("HTTP сервер начал работу")
	err := srv.ListenAndServe()
	<-idleConnsClosed
	return err
}
