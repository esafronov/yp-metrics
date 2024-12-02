package agent

import (
	"context"
	"reflect"
	"runtime"
	"testing"

	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestAgent_UpdateMetrics(t *testing.T) {

	type want struct {
		metricName storage.MetricName
		gvalue     float64
		cvalue     int64
	}
	tests := []struct {
		name    string
		a       *Agent
		want    want
		metrics []storage.Metrics
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
			name:      "Read Alloc param positive",
			a:         &Agent{},
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
