package agent

import (
	"context"
	"reflect"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/shirou/gopsutil/mem"
	"github.com/stretchr/testify/require"
)

func TestAgent_UpdateMetrics(t *testing.T) {

	type want struct {
		metricName storage.MetricName
		gvalue     float64
		cvalue     int64
	}
	tests := []struct {
		a       *Agent
		name    string
		metrics []storage.Metrics
		want    want
	}{
		{
			name: "Save TotalAlloc=0.01 to storage positive",
			a: &Agent{
				storage:  storage.NewMemStorage(),
				chUpdate: make(chan storage.Metrics),
			},
			metrics: []storage.Metrics{
				{
					ID:          "TotalAlloc",
					MType:       "gauge",
					ActualValue: float64(0.01),
				},
			},
			want: want{
				metricName: storage.MeticNameTotalAlloc,
				gvalue:     float64(0.01),
			},
		},
		{
			name: "Update Lookups=123 in storage to Lookups=456",
			a: &Agent{
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"Lookups": storage.NewMetricGauge(float64(123)),
					},
				},
				chUpdate: make(chan storage.Metrics),
			},
			metrics: []storage.Metrics{
				{
					ID:          "Lookups",
					MType:       "gauge",
					ActualValue: float64(456),
				},
			},
			want: want{
				metricName: storage.MeticNameLookups,
				gvalue:     float64(456),
			},
		},
		{
			name: "Update PollCount=1 in storage to PollCount=2",
			a: &Agent{
				memStats: runtime.MemStats{Lookups: uint64(456)},
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"PollCount": storage.NewMetricCounter(int64(1)),
					},
				},
				chUpdate: make(chan storage.Metrics),
			},
			metrics: []storage.Metrics{
				{
					ID:          "PollCount",
					MType:       "counter",
					ActualValue: int64(1),
				},
			},
			want: want{
				metricName: storage.MetricNamePollCount,
				cvalue:     int64(2),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tt.a.UpdateMetrics(ctx)
			for _, m := range tt.metrics {
				tt.a.chUpdate <- m
			}
			close(tt.a.chUpdate)
			time.Sleep(time.Duration(1) * time.Second)
			m, err := tt.a.storage.Get(ctx, tt.want.metricName)
			require.NoError(t, err, "Получена ошибка")
			require.NotNil(t, m, "Метрика не найдена в хранилище по ключу %s", tt.want.metricName)
			mv := m.GetValue()
			switch m.(type) {
			case *storage.MetricCounter:
				require.Equal(t, tt.want.cvalue, mv.(int64), "Метрика имеет отличное значение от ожидаемого")
			case *storage.MetricGauge:
				require.Equal(t, tt.want.gvalue, mv.(float64), "Метрика имеет отличное значение от ожидаемого")
			}
		})
	}

}

func TestAgent_ReadStat(t *testing.T) {
	tests := []struct {
		name      string
		a         *Agent
		wantParam string
	}{
		{
			name: "Read Alloc param positive",
			a: &Agent{
				memReadFunc: runtime.ReadMemStats,
			},
			wantParam: "Alloc",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.a.ReadStat()
			r := reflect.ValueOf(tt.a.memStats)
			rv := reflect.Indirect(r).FieldByName(tt.wantParam)
			if !rv.CanUint() {
				t.Errorf("параметр %s не считан из runtime", tt.wantParam)
			}
		})
	}
}

func TestAgent_collectMemStat(t *testing.T) {
	pollInterval := 1
	var testValue float64 = 101

	tests := []struct {
		name string
		want storage.Metrics
	}{
		{
			name: "collect Alloc param",
			want: storage.Metrics{
				ID:          "Alloc",
				MType:       "gauge",
				ActualValue: testValue,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			a := &Agent{
				memReadFunc: func(m *runtime.MemStats) {
					m.Alloc = uint64(101)
				},
			}
			ch := a.collectMemStat(ctx, &pollInterval)
			time.Sleep(time.Duration(1500) * time.Millisecond)
			cancel()
			var got []storage.Metrics
			for m := range ch {
				got = append(got, m)
			}
			if !slices.Contains(got, tt.want) {
				t.Errorf("wanted metric %v has not been received from channel", tt.want)
			}
		})
	}
}

func TestAgent_collectExtraStat(t *testing.T) {
	pollInterval := 1
	var testValue float64 = 99
	var testValue2 float64 = 20
	tests := []struct {
		name  string
		want  storage.Metrics
		want2 storage.Metrics
	}{
		{
			name: "collect TotalMemory param",
			want: storage.Metrics{
				ID:          "TotalMemory",
				MType:       "gauge",
				ActualValue: testValue,
			},
			want2: storage.Metrics{
				ID:          "CPUutilization1",
				MType:       "gauge",
				ActualValue: testValue2,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			a := &Agent{
				vmemReadFunc: func() (*mem.VirtualMemoryStat, error) {
					return &mem.VirtualMemoryStat{
						Total: uint64(99),
					}, nil
				},
				cpuReadFunc: func(interval time.Duration, percpu bool) ([]float64, error) {
					return []float64{20}, nil
				},
			}
			ch := a.collectExtraStat(ctx, &pollInterval)
			time.Sleep(time.Duration(1500) * time.Millisecond)
			cancel()
			var got []storage.Metrics
			for m := range ch {
				got = append(got, m)
			}
			if !slices.Contains(got, tt.want) {
				t.Errorf("required metric %v has not been received from channel", tt.want)
			}
			if !slices.Contains(got, tt.want2) {
				t.Errorf("required metric %v has not been received from channel", tt.want2)
			}
		})
	}
}

func TestAgent_collectMetrics(t *testing.T) {
	pollInterval := 1
	var testValue float64 = 99
	var testValue2 float64 = 20
	tests := []struct {
		name  string
		want  storage.Metrics
		want2 storage.Metrics
	}{
		{
			name: "collect different params",
			want: storage.Metrics{
				ID:          "TotalMemory",
				MType:       "gauge",
				ActualValue: testValue,
			},
			want2: storage.Metrics{
				ID:          "Alloc",
				MType:       "gauge",
				ActualValue: testValue2,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			a := &Agent{
				memReadFunc: func(m *runtime.MemStats) {
					m.Alloc = uint64(20)
				},
				vmemReadFunc: func() (*mem.VirtualMemoryStat, error) {
					return &mem.VirtualMemoryStat{
						Total: uint64(99),
					}, nil
				},
				cpuReadFunc: func(interval time.Duration, percpu bool) ([]float64, error) {
					return []float64{20}, nil
				},
				chUpdate: make(chan storage.Metrics),
			}
			a.CollectMetrics(ctx, &pollInterval)
			time.Sleep(time.Duration(1500) * time.Millisecond)
			cancel()
			var got []storage.Metrics
			for m := range a.chUpdate {
				got = append(got, m)
			}
			if !slices.Contains(got, tt.want) {
				t.Errorf("required metric %v has not been received from channel", tt.want)
			}
			if !slices.Contains(got, tt.want2) {
				t.Errorf("required metric %v has not been received from channel", tt.want2)
			}
		})
	}
}
