// Agent package collects cpu, memory metrics on local machine and send them to Server app by HTTP
//
// Main functions are : Run, ReadStat, CollectMetrics, UpdateMetrics, SendMetrics
package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"runtime"

	_ "net/http/pprof" // подключаем пакет pprof

	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/pprofserv"
	"github.com/esafronov/yp-metrics/internal/storage"
)

type Agent struct {
	storage       storage.Repositories //storage for metrics
	serverAddress string               //server address to send report
	memStats      runtime.MemStats     //mem stat
	chUpdate      chan storage.Metrics //channel for metrics should be updated in repository
	chSend        chan storage.Metrics //channel for metrics should be send to server
}

// Fabric function
func NewAgent(s storage.Repositories, serverAddress string) *Agent {
	return &Agent{
		storage:       s,
		serverAddress: serverAddress,
		chUpdate:      make(chan storage.Metrics, 20),
		chSend:        make(chan storage.Metrics, 20),
	}
}

var serverAddress *string        //server address
var pollInterval *int            //interval to poll metrics
var reportInterval *int          //send report interval
var secretKey *string            //secretKey
var rateLimit *int               //parallel request limit
var profileServerAddress *string //profile serveraddress to listen

// Make initialization and run main buisness logic:
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
	if *profileServerAddress != "" {
		profileServer := pprofserv.NewDebugServer(*profileServerAddress)
		profileServer.Start()
		defer profileServer.Close()
	}
	a := NewAgent(storage.NewMemStorage(), "http://"+*serverAddress)
	ctx, cancel := context.WithCancel(context.Background())
	a.CollectMetrics(ctx, pollInterval)
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
		s := <-sigs
		fmt.Println("got signal ", s)
		cancel()
	}()
	fmt.Println("report to:", *serverAddress)
	fmt.Println("report interval:", *reportInterval)
	fmt.Println("poll interval:", *pollInterval)
	fmt.Println("key:", *secretKey)
	fmt.Println("rate limit:", *rateLimit)
	a.UpdateMetrics(ctx)
	a.SendMetrics(ctx, reportInterval, rateLimit)
	fmt.Println("exit")
}
