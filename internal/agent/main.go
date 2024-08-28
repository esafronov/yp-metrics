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
			a.storage.Update(exists, v)
		} else {
			a.storage.Insert(m, storage.NewMetricGauge(v))
		}
	}

	if existed := a.storage.Get(storage.MetricNamePollCount); existed != nil {
		a.storage.Update(existed, int64(1))
	} else {
		a.storage.Insert(storage.MetricNamePollCount, storage.NewMetricCounter(int64(1)))
	}

	rn := rand.New(rand.NewSource(time.Now().UnixNano()))
	if existed := a.storage.Get(storage.MetricNameRandomValue); existed != nil {
		a.storage.Update(existed, rn.Float64())
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
		res, err := http.Post(url, "application/json", bytes.NewReader(marshaled))
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
		duration := time.Since(timeStamp)
		if duration.Seconds() >= float64(reportInterval) {
			timeStamp = time.Now()
			if err := a.SendReport(); err != nil {
				fmt.Println(err.Error())
				continue
			}
		}
	}
}
