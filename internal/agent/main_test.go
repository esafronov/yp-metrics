package agent

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"

	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/stretchr/testify/require"
)

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

func TestAgent_StoreStat(t *testing.T) {
	type want struct {
		metricName storage.MetricName
		gvalue     float64
		cvalue     int64
	}
	tests := []struct {
		name string
		a    *Agent
		want want
	}{
		{
			name: "Save TotalAlloc=123 to storage positive",
			a: &Agent{
				memStats: runtime.MemStats{TotalAlloc: uint64(123)},
				storage:  storage.NewMemStorage(),
			},
			want: want{
				metricName: storage.MeticNameTotalAlloc,
				gvalue:     float64(123),
			},
		},
		{
			name: "Update Lookups=123 in storage to Lookups=456",
			a: &Agent{
				memStats: runtime.MemStats{Lookups: uint64(456)},
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"Lookups": storage.NewMetricGauge(float64(123)),
					},
				},
			},
			want: want{
				metricName: storage.MeticNameLookups,
				gvalue:     float64(456),
			},
		},
		{
			name: "Update Lookups=123 in storage to Lookups=456",
			a: &Agent{
				memStats: runtime.MemStats{Lookups: uint64(456)},
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"Lookups": storage.NewMetricGauge(float64(123)),
					},
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
						"Lookups":   storage.NewMetricGauge(float64(123)),
					},
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
			tt.a.StoreStat()
			m := tt.a.storage.Get(tt.want.metricName)
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

func TestAgent_SendReport(t *testing.T) {
	type want struct {
		contentType string
		request     string
	}

	tests := []struct {
		name string
		a    *Agent
		want want
	}{
		{
			name: "send /update/gauge/Lookups/1.200000",
			a: &Agent{
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"Lookups": storage.NewMetricGauge(float64(1.200000)),
					},
				},
			},
			want: want{
				contentType: "text/plain",
				request:     "/update/gauge/Lookups/1.200000",
			},
		},
		{
			name: "send /update/counter/PollCount/1",
			a: &Agent{
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"PollCount": storage.NewMetricCounter(int64(1)),
					},
				},
			},
			want: want{
				contentType: "text/plain",
				request:     "/update/counter/PollCount/1",
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				require.Equal(t, tt.want.contentType, req.Header.Get("Content-Type"))
				require.Equal(t, tt.want.request, req.URL.String())
				//rw.Write([]byte(`OK`))
			}))
			// Close the server when test finishes
			defer server.Close()
			tt.a.serverAddress = server.URL
			tt.a.SendReport()
		})
	}
}
