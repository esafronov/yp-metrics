package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"reflect"
	"runtime"

	"github.com/esafronov/yp-metrics/internal/compress"
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
	for _, metricName := range storage.GetGaugeMetrics() {
		rv := reflect.Indirect(r).FieldByName(string(metricName))
		var v interface{}
		if rv.CanUint() {
			v = float64(rv.Uint())
		} else if rv.CanFloat() {
			v = rv.Float()
		}
		if exists := a.storage.Get(metricName); exists != nil {
			a.storage.Update(metricName, v)
		} else {
			a.storage.Insert(metricName, storage.NewMetricGauge(v))
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

func (a *Agent) SendReport() error {
	var reqMetric *storage.Metrics
	for metricName, v := range a.storage.GetAll() {
		url := a.serverAddress + "/update/"
		reqMetric = &storage.Metrics{
			ID:          string(metricName),
			ActualValue: v.GetValue(),
		}
		marshaled, err := json.Marshal(reqMetric)
		if err != nil {
			return fmt.Errorf("marshal error %s", err)
		}
		var data bytes.Buffer
		err = compress.GzipToBuffer(marshaled, &data)
		if err != nil {
			return fmt.Errorf("failed compress request %s", err)
		}
		req, err := http.NewRequest(http.MethodPost, url, &data)
		if err != nil {
			return fmt.Errorf("post request: %s", err)
		}
		//header Accept-Encoding : gzip will be added automatically, so not need to add
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("post request: %s", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("response status: %d", res.StatusCode)
		}
	}
	return nil
}

var serverAddress string
var pollInterval int = -1
var reportInterval int = -1

func Run() {
	if err := parseEnv(); err != nil {
		fmt.Printf("env parse err %v\n", err)
		return
	}
	parseFlags()
	a := &Agent{
		storage:       storage.NewMemStorage(),
		serverAddress: "http://" + serverAddress,
	}
	timeStamp := time.Now()
	for {
		time.Sleep(time.Duration(pollInterval) * time.Second)
		a.ReadStat()
		a.StoreStat()
		if time.Since(timeStamp).Seconds() >= float64(reportInterval) {
			timeStamp = time.Now()
			if err := a.SendReport(); err != nil {
				fmt.Println(err.Error())
				continue
			}
		}
	}
}
