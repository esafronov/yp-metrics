package agent

import (
	"bytes"
	"context"
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
	ctx := context.Background()
	for _, metricName := range storage.GetGaugeMetrics() {
		rv := reflect.Indirect(r).FieldByName(string(metricName))
		var v interface{}
		if rv.CanUint() {
			v = float64(rv.Uint())
		} else if rv.CanFloat() {
			v = rv.Float()
		}
		metric, _ := a.storage.Get(ctx, metricName)
		if metric != nil {
			a.storage.Update(ctx, metricName, v, metric)
		} else {
			a.storage.Insert(ctx, metricName, storage.NewMetricGauge(v))
		}
	}
	metric, _ := a.storage.Get(ctx, storage.MetricNamePollCount)
	if metric != nil {
		a.storage.Update(ctx, storage.MetricNamePollCount, int64(1), metric)
	} else {
		a.storage.Insert(ctx, storage.MetricNamePollCount, storage.NewMetricCounter(int64(1)))
	}

	rn := rand.New(rand.NewSource(time.Now().UnixNano()))
	metric, _ = a.storage.Get(ctx, storage.MetricNameRandomValue)
	if metric != nil {
		a.storage.Update(ctx, storage.MetricNameRandomValue, rn.Float64(), metric)
	} else {
		a.storage.Insert(ctx, storage.MetricNameRandomValue, storage.NewMetricGauge(rn.Float64()))
	}
}

func (a *Agent) SendReport() error {
	var reqMetric *storage.Metrics
	items, err := a.storage.GetAll(context.Background())
	if err != nil {
		return fmt.Errorf("cannot get metrics %s", err)
	}
	for metricName, v := range items {
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
var pollInterval *int
var reportInterval *int

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
		time.Sleep(time.Duration(*pollInterval) * time.Second)
		a.ReadStat()
		a.StoreStat()
		if time.Since(timeStamp).Seconds() >= float64(*reportInterval) {
			timeStamp = time.Now()
			if err := a.SendReport(); err != nil {
				fmt.Println(err.Error())
				continue
			}
		}
	}
}
