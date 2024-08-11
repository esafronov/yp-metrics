package agent

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"reflect"
	"runtime"

	"github.com/esafronov/yp-metrics/internal/storage"
)

type Agent struct {
	storage       storage.Repositories
	serverAddress string
	memStats      runtime.MemStats
}

func (a *Agent) ReadStat() {
	runtime.ReadMemStats(&a.memStats)
}

func (a *Agent) StoreStat() {
	r := reflect.ValueOf(a.memStats)
	for _, m := range storage.GetGaugeMetrics() {
		rv := reflect.Indirect(r).FieldByName(string(m))
		var v interface{}
		if rv.CanUint() {
			v = float64(rv.Uint())
		} else if rv.CanFloat() {
			v = rv.Float()
		}
		if exists := a.storage.Get(m); exists != nil {
			a.storage.Update(m, v)
		} else {
			a.storage.Insert(m, storage.NewMetricGauge(v))
		}
	}

	if exists := a.storage.Get(storage.MetricNamePollCount); exists != nil {
		a.storage.Update(storage.MetricNamePollCount, int64(1))
	} else {
		a.storage.Insert(storage.MetricNamePollCount, storage.NewMetricCounter(int64(1)))
	}

	rn := rand.New(rand.NewSource(time.Now().UnixNano()))
	if exists := a.storage.Get(storage.MetricNameRandomValue); exists != nil {
		a.storage.Update(storage.MetricNameRandomValue, rn.Float64())
	} else {
		a.storage.Insert(storage.MetricNameRandomValue, storage.NewMetricGauge(rn.Float64()))
	}
}

func (a *Agent) SendReport() {
	for mn, v := range a.storage.GetAll() {
		url := a.serverAddress + "/update/"
		switch v.(type) {
		case *storage.MetricGauge:
			mv, _ := v.GetValue().(float64)
			url += string(storage.MetricTypeGauge) + "/" + string(mn) + "/" + fmt.Sprintf("%f", mv)
		case *storage.MetricCounter:
			mv, _ := v.GetValue().(int64)
			url += string(storage.MetricTypeCounter) + "/" + string(mn) + "/" + fmt.Sprint(mv)
		}
		var ioReader io.Reader
		res, err := http.Post(url, "text/plain", ioReader)
		if err != nil {
			panic("error sending request")
		}
		if res.StatusCode != http.StatusOK {
			panic("status is not 200 OK")
		}
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			panic("error reading response 200 OK")
		}
		bodyStr := string(resBody)
		if bodyStr != "" {
			panic("body should be empty")
		}
		res.Body.Close()
	}
}

func Run() {
	parseFlags()
	a := &Agent{
		storage:       storage.NewMemStorage(),
		serverAddress: "http://" + flagServerAddress,
	}
	timeStamp := time.Now()
	for {
		time.Sleep(time.Duration(flagPollInterval) * time.Second)
		a.ReadStat()
		a.StoreStat()
		duration := time.Since(timeStamp)
		if duration.Seconds() >= float64(flagReportInterval) {
			timeStamp = time.Now()
			a.SendReport()
		}
	}
}
