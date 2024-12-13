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

	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/pprofserv"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
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
}

// NewAgent is fabric method
func NewAgent(s storage.Repositories, serverAddress string) *Agent {
	return &Agent{
		storage:       s,
		serverAddress: serverAddress,
		chUpdate:      make(chan storage.Metrics, 20),
		chSend:        make(chan storage.Metrics, 20),
		memReadFunc:   runtime.ReadMemStats,
		vmemReadFunc:  mem.VirtualMemory,
		cpuReadFunc:   cpu.Percent,
	}
}

func (a *Agent) setMemReadFunc(memReadFunc func(m *runtime.MemStats)) {
	a.memReadFunc = memReadFunc
}

func (a *Agent) setVMemReadFunc(vmemReadFunc func() (*mem.VirtualMemoryStat, error)) {
	a.vmemReadFunc = vmemReadFunc
}

func (a *Agent) setCpuReadFunc(cpuReadFunc func(interval time.Duration, percpu bool) ([]float64, error)) {
	a.cpuReadFunc = cpuReadFunc
}

var serverAddress *string        //server address
var pollInterval *int            //interval to poll metrics
var reportInterval *int          //send report interval
var secretKey *string            //secretKey
var rateLimit *int               //parallel request limit
var profileServerAddress *string //profile serveraddress to listen

// Run initialize and run main buisness logic:
//
// Get env/flags params, initialize repository, runs routine for collecting and sending metrics
func Run() {
	if err := parseEnv(); err != nil {
		fmt.Printf("env parse err %v\n", err)
		return
	}
	parseFlags()
	err := logger.Initialize("debug")
	if err != nil {
		fmt.Println("can't init logger", err)
		return
	}
	//run profile server if env/flag is set
	if profileServerAddress != nil && *profileServerAddress != "" {
		profileServer := pprofserv.NewDebugServer(*profileServerAddress)
		profileServer.Start()
		defer profileServer.Close()
	}
	if serverAddress == nil {
		panic("serverAddress is nil")
	}
	a := NewAgent(storage.NewMemStorage(), "http://"+*serverAddress)
	a.setMemReadFunc(runtime.ReadMemStats)
	a.setCpuReadFunc(cpu.Percent)
	a.setVMemReadFunc(mem.VirtualMemory)
	ctx, cancel := context.WithCancel(context.Background())
	if pollInterval == nil {
		panic("pollInterval is null")
	}
	a.CollectMetrics(ctx, pollInterval)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
		s := <-sigs
		fmt.Println("got signal ", s)
		cancel()
	}()
	if serverAddress != nil {
		fmt.Println("report to:", *serverAddress)
	}
	if reportInterval != nil {
		fmt.Println("report interval:", *reportInterval)
	}
	if pollInterval != nil {
		fmt.Println("poll interval:", *pollInterval)
	}
	if secretKey != nil {
		fmt.Println("key:", *secretKey)
	}
	if rateLimit != nil {
		fmt.Println("rate limit:", *rateLimit)
	}
	a.UpdateMetrics(ctx)
	a.SendMetrics(ctx, reportInterval, rateLimit)
	fmt.Println("exit")
}
