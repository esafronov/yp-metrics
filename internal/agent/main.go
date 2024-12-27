// Package agent collects cpu and memory stat on local machine then send it to Server app by HTTP
//
// Main functions are : Run, ReadStat, CollectMetrics, UpdateMetrics, SendMetrics
package agent

import (
	"context"
	"fmt"
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

func OptionWithMemReadFunc(memReadFunc func(m *runtime.MemStats)) func(a *Agent) {
	return func(a *Agent) {
		a.memReadFunc = memReadFunc
	}
}

func OptionWithVMemReadFunc(vmemReadFunc func() (*mem.VirtualMemoryStat, error)) func(a *Agent) {
	return func(a *Agent) {
		a.vmemReadFunc = vmemReadFunc
	}
}

func OptionWithCpuReadFunc(cpuReadFunc func(interval time.Duration, percpu bool) ([]float64, error)) func(a *Agent) {
	return func(a *Agent) {
		a.cpuReadFunc = cpuReadFunc
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
	params := config.Params
	logger.Log.Info("params",
		zap.String("Address", *params.Address),
		zap.Int("ReportInterval", *params.ReportInterval),
		zap.Int("PollInterval", *params.PollInterval),
		zap.Int("RateLimit", *params.RateLimit),
		zap.String("ProfileServerAddress", *params.ProfileServerAddress),
		zap.String("SecretKey", *params.SecretKey),
		zap.String("CryptoKey", *params.CryptoKey),
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
	a := NewAgent(
		storage.NewMemStorage(),
		"http://"+*params.Address,
		OptionWithSecretKey(*params.SecretKey),
		OptionWithCryptoKey(*params.CryptoKey),
		OptionWithCpuReadFunc(cpu.Percent),
		OptionWithMemReadFunc(runtime.ReadMemStats),
		OptionWithVMemReadFunc(mem.VirtualMemory),
	)
	ctx, cancel := context.WithCancel(context.Background())
	if params.PollInterval == nil {
		panic("pollInterval is null")
	}
	a.CollectMetrics(ctx, params.PollInterval)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
		s := <-sigs
		fmt.Println("got signal ", s)
		cancel()
	}()
	a.UpdateMetrics(ctx)
	a.SendMetrics(ctx, params.ReportInterval, params.RateLimit)
	fmt.Println("exit")
}
