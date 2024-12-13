package agent

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"reflect"

	"github.com/esafronov/yp-metrics/internal/storage"
)

// Routine for collecting general metrics, returns channel for reading
func (a *Agent) collectMemStat(ctx context.Context, pollInterval *int) chan storage.Metrics {
	ch := make(chan storage.Metrics, 20)
	ticker := time.NewTicker(time.Duration(*pollInterval) * time.Second)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.ReadStat()
				r := reflect.ValueOf(a.memStats)
				for _, metricName := range storage.GetGaugeMetrics() {
					rv := r.FieldByName(string(metricName))
					if !rv.IsValid() {
						continue
					}
					var v interface{}
					if rv.CanUint() {
						v = float64(rv.Uint())
					} else if rv.CanFloat() {
						v = rv.Float()
					}
					ch <- storage.Metrics{
						ID:          string(metricName),
						MType:       string(storage.MetricTypeGauge),
						ActualValue: v,
					}
				}
				ch <- storage.Metrics{
					ID:          string(storage.MetricNamePollCount),
					MType:       string(storage.MetricTypeCounter),
					ActualValue: int64(1),
				}
				rn := rand.New(rand.NewSource(time.Now().UnixNano()))
				ch <- storage.Metrics{
					ID:          string(storage.MetricNameRandomValue),
					MType:       string(storage.MetricTypeGauge),
					ActualValue: rn.Float64(),
				}
			}
		}
	}()
	return ch
}

// Routine for collecting extra gauge metrics, returns channel for reading them
func (a *Agent) collectExtraStat(ctx context.Context, pollInterval *int) chan storage.Metrics {
	ch := make(chan storage.Metrics, 20)
	ticker := time.NewTicker(time.Duration(*pollInterval) * time.Second)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				vmem, err := a.vmemReadFunc()
				if err != nil {
					panic("Error when reading mem")
				}
				ch <- storage.Metrics{
					ID:          string(storage.MetricNameTotalMemory),
					MType:       string(storage.MetricTypeGauge),
					ActualValue: float64(vmem.Total),
				}
				ch <- storage.Metrics{
					ID:          string(storage.MetricNameFreeMemory),
					MType:       string(storage.MetricTypeGauge),
					ActualValue: float64(vmem.Free),
				}
				vcpu, err := a.cpuReadFunc(0, false)
				if err != nil {
					panic("Error when reading cpu")
				}
				ch <- storage.Metrics{
					ID:          string(storage.MetricNameCPUutilization1),
					MType:       string(storage.MetricTypeGauge),
					ActualValue: float64(vcpu[0]),
				}
			}
		}
	}()
	return ch
}

// CollectMetrics run 2 routines for unit collected metrics from two channels into one
func (a *Agent) CollectMetrics(ctx context.Context, pollInterval *int) {
	var wg sync.WaitGroup
	wg.Add(2)
	processCh := func(c chan storage.Metrics) {
		for data := range c {
			a.chUpdate <- data
		}
		wg.Done()
	}
	go processCh(a.collectMemStat(ctx, pollInterval))
	go processCh(a.collectExtraStat(ctx, pollInterval))
	go func() {
		wg.Wait()
		close(a.chUpdate)
	}()
}

// UpdateMetrics run routine for reading metrics from channel for updating in repository
func (a *Agent) UpdateMetrics(ctx context.Context) {
	go func() {
		for m := range a.chUpdate {
			metric, _ := a.storage.Get(ctx, storage.MetricName(m.ID))
			if metric != nil {
				err := a.storage.Update(ctx, storage.MetricName(m.ID), m.ActualValue, metric)
				if err != nil {
					panic(err.Error())
				}
			} else {
				switch m.MType {
				case string(storage.MetricTypeCounter):
					err := a.storage.Insert(ctx, storage.MetricName(m.ID), storage.NewMetricCounter(m.ActualValue))
					if err != nil {
						panic(err.Error())
					}
				case string(storage.MetricTypeGauge):
					err := a.storage.Insert(ctx, storage.MetricName(m.ID), storage.NewMetricGauge(m.ActualValue))
					if err != nil {
						panic(err.Error())
					}
				}
			}
		}
	}()
}

// ReadStat read system metrics into structure
func (a *Agent) ReadStat() {
	a.memReadFunc(&a.memStats)
}
