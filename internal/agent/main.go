// Package agent collects cpu and memory stat on local machine then send it to Server app by HTTP
//
// Main functions are : Run, ReadStat, CollectMetrics, UpdateMetrics, SendMetrics
package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"runtime"

	_ "net/http/pprof" // подключаем пакет pprof

	"github.com/esafronov/yp-metrics/internal/agent/config"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/pprofserv"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	// импортируем пакет со сгенерированными protobuf-файлами
	pb "github.com/esafronov/yp-metrics/internal/grpc/proto"
)

type Agent struct {
	storage       storage.Repositories
	chUpdate      chan storage.Metrics
	chSend        chan storage.Metrics
	memReadFunc   func(m *runtime.MemStats)
	vmemReadFunc  func() (*mem.VirtualMemoryStat, error)
	cpuReadFunc   func(interval time.Duration, percpu bool) ([]float64, error)
	serverAddress string
	memStats      runtime.MemStats
	secretKey     string
	cryptoKey     string
	metricsClient pb.MetricsClient
}

// NewAgent is fabric method
func NewAgent(s storage.Repositories, serverAddress string, opts ...func(a *Agent)) *Agent {
	a := &Agent{
		storage:       s,
		serverAddress: serverAddress,
		chUpdate:      make(chan storage.Metrics, 20),
		chSend:        make(chan storage.Metrics, 20),
		memReadFunc:   runtime.ReadMemStats,
		vmemReadFunc:  mem.VirtualMemory,
		cpuReadFunc:   cpu.Percent,
	}
	for _, f := range opts {
		f(a)
	}
	return a
}

// OptionWithSecretKey option function to configure Agent to use secretKey
func OptionWithSecretKey(secretKey string) func(a *Agent) {
	return func(a *Agent) {
		a.secretKey = secretKey
	}
}

// OptionWithCryptoKey option function to configure Agent to use cryptoKey
func OptionWithCryptoKey(cryptoKey string) func(a *Agent) {
	return func(a *Agent) {
		a.cryptoKey = cryptoKey
	}
}

// OptionWithMemReadFunc option function to configure Agent to use MemReadFunc
func OptionWithMemReadFunc(memReadFunc func(m *runtime.MemStats)) func(a *Agent) {
	return func(a *Agent) {
		a.memReadFunc = memReadFunc
	}
}

// OptionWithVMemReadFunc option function to configure Agent to use MemReadFunc
func OptionWithVMemReadFunc(vmemReadFunc func() (*mem.VirtualMemoryStat, error)) func(a *Agent) {
	return func(a *Agent) {
		a.vmemReadFunc = vmemReadFunc
	}
}

// OptionWithCpuReadFunc option function to configure Agent to use CpuReadFunc
func OptionWithCpuReadFunc(cpuReadFunc func(interval time.Duration, percpu bool) ([]float64, error)) func(a *Agent) {
	return func(a *Agent) {
		a.cpuReadFunc = cpuReadFunc
	}
}

// OptionWithMetricsClient option function to configure Agent to use metricsClient
func OptionWithMetricsClient(client pb.MetricsClient) func(a *Agent) {
	return func(a *Agent) {
		if client == nil {
			return
		}
		a.metricsClient = client
	}
}

// Run initialize and run main buisness logic:
//
// Get env/flags params, initialize repository, runs routine for collecting and sending metrics
func Run() {
	err := logger.Initialize("debug")
	if err != nil {
		fmt.Println("can't init logger", err)
		return
	}
	config.Initialize()
	params := config.Params
	logger.Log.Info("params",
		zap.String("Address", *params.Address),
		zap.Int("ReportInterval", *params.ReportInterval),
		zap.Int("PollInterval", *params.PollInterval),
		zap.Int("RateLimit", *params.RateLimit),
		zap.String("ProfileServerAddress", *params.ProfileServerAddress),
		zap.String("SecretKey", *params.SecretKey),
		zap.String("CryptoKey", *params.CryptoKey),
		zap.Bool("UseGRPC", *params.UseGRPC),
		zap.String("Config", *params.Config),
	)
	//run profile server if env/flag is set
	if params.ProfileServerAddress != nil && *params.ProfileServerAddress != "" {
		profileServer := pprofserv.NewDebugServer(*params.ProfileServerAddress)
		profileServer.Start()
		defer profileServer.Close()
	}
	if params.Address == nil {
		panic("serverAddress is nil")
	}
	var metricsClient pb.MetricsClient
	if *params.UseGRPC {
		var creds credentials.TransportCredentials
		if *params.CryptoKey != "" {
			fmt.Println("CryptoKey", *params.CryptoKey)
			// Create tls based credential.
			creds, err = credentials.NewClientTLSFromFile(*params.CryptoKey, "")
			if err != nil {
				log.Fatalf("failed to load credentials: %v", err)
				return
			}
		} else {
			creds = insecure.NewCredentials()
		}
		conn, err := grpc.NewClient(*params.Address, grpc.WithTransportCredentials(creds))
		if err != nil {
			fmt.Println("can't establish connection to grpc server", err)
			return
		}
		metricsClient = pb.NewMetricsClient(conn)
		defer conn.Close()
	}
	a := NewAgent(
		storage.NewMemStorage(),
		"http://"+*params.Address,
		OptionWithSecretKey(*params.SecretKey),
		OptionWithCryptoKey(*params.CryptoKey),
		OptionWithCpuReadFunc(cpu.Percent),
		OptionWithMemReadFunc(runtime.ReadMemStats),
		OptionWithVMemReadFunc(mem.VirtualMemory),
		OptionWithMetricsClient(metricsClient),
	)
	ctx, cancel := context.WithCancel(context.Background())
	if params.PollInterval == nil {
		panic("pollInterval is null")
	}
	a.CollectMetrics(ctx, params.PollInterval)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGQUIT)
	go func() {
		s := <-sigs
		fmt.Println("got signal ", s)
		cancel()
	}()
	a.UpdateMetrics(ctx)
	a.SendMetrics(ctx, params.ReportInterval, params.RateLimit)
	fmt.Println("exit")
}
