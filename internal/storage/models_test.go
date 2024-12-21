// Package storage includes repository implementations: MemStorage, HybridStorage, DbStorage, DTO objects and entities

package storage

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetrics_UnmarshalJSON(t *testing.T) {
	type args struct {
		data []byte
	}
	var counterValue int64 = 1
	gaugeValue := 0.001
	tests := []struct {
		name    string
		want    Metrics
		args    args
		wantErr bool
	}{
		{
			name: "positive counter",
			want: Metrics{
				ID:          "test",
				MType:       "counter",
				Delta:       &counterValue,
				ActualValue: &counterValue,
			},
			args: args{
				data: []byte(`{
					"id":"test",
					"type":"counter",
					"delta":1	
				}`),
			},
			wantErr: false,
		},
		{
			name: "positive gauge",
			want: Metrics{
				ID:          "test2",
				MType:       "gauge",
				Value:       &gaugeValue,
				ActualValue: &gaugeValue,
			},
			args: args{
				data: []byte(`{
					"id":"test2",
					"type":"gauge",
					"value":0.001
				}`),
			},
			wantErr: false,
		},
		{
			name: "metric type is unknown",
			want: Metrics{
				ID:          "test2",
				MType:       "gauge",
				Value:       &gaugeValue,
				ActualValue: &gaugeValue,
			},
			args: args{
				data: []byte(`{
					"id":"test2",
					"type":"unknown",
					"value":0.001
				}`),
			},
			wantErr: true,
		},
		{
			name: "wrong json format",
			want: Metrics{
				ID:          "test2",
				MType:       "gauge",
				Value:       &gaugeValue,
				ActualValue: &gaugeValue,
			},
			args: args{
				data: []byte(`{
					"id":"
				}`),
			},
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Metrics
			if err := m.UnmarshalJSON(tt.args.data); (err != nil) != tt.wantErr {
				if err == nil {
					require.Equal(t, tt.want, m)
				}
				t.Errorf("Metrics.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetrics_MarshalJSON(t *testing.T) {
	var counterValue int64 = 1
	gaugeValue := 0.001
	wrongValue := "wrongvaluetype"
	tests := []struct {
		name    string
		m       Metrics
		want    []byte
		wantErr bool
	}{
		{
			name: "positive counter",
			m: Metrics{
				ID:          "test",
				ActualValue: counterValue,
			},
			want: []byte(`{
					"id":"test",
					"type":"counter",
					"delta":1	
				}`),
			wantErr: false,
		},
		{
			name: "positive gauge",
			m: Metrics{
				ID:          "test2",
				ActualValue: gaugeValue,
			},
			want: []byte(`{
					"id":"test2",
					"type":"gauge",
					"value":0.001
				}`),
			wantErr: false,
		},
		{
			name: "wrong ActualValue type",
			m: Metrics{
				ID:          "test2",
				ActualValue: wrongValue,
			},
			want: []byte(`{
					"id":"test2",
					"type":"gauge",
					"value":0.001
				}`),
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.m.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Metrics.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				require.JSONEq(t, string(tt.want), string(got))
			}
		})
	}
}

func TestMetricGauge_String(t *testing.T) {
	tests := []struct {
		name string
		m    *MetricGauge
		want string
	}{
		{
			name: "gauge as string",
			m: &MetricGauge{
				val: float64(0.001),
			},
			want: "0.001",
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.String(); got != tt.want {
				t.Errorf("MetricGauge.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetricCounter_String(t *testing.T) {
	tests := []struct {
		name string
		m    *MetricCounter
		want string
	}{
		{
			name: "counter as string",
			m: &MetricCounter{
				val: int64(1),
			},
			want: "1",
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.String(); got != tt.want {
				t.Errorf("MetricCounter.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGaugeMetrics(t *testing.T) {
	tests := []struct {
		name string
		want []MetricName
	}{
		{
			name: "get gauge metrics",
			want: []MetricName{
				MetricNameAlloc,
				MetricNameBuckHashSys,
				MetricNameBuckFrees,
				MeticNameGCCPUFraction,
				MeticNameGCSys,
				MeticNameHeapAlloc,
				MeticNameHeapIdle,
				MeticNameHeapInuse,
				MeticNameHeapObjects,
				MeticNameHeapReleased,
				MeticNameHeapSys,
				MeticNameLastGC,
				MeticNameLookups,
				MeticNameMCacheInuse,
				MeticNameMCacheSys,
				MeticNameMSpanInuse,
				MeticNameMSpanSys,
				MeticNameMallocs,
				MeticNameNextGC,
				MeticNameNumForcedGC,
				MeticNameNumGC,
				MeticNameOtherSys,
				MeticNamePauseTotalNs,
				MeticNameStackInuse,
				MeticNameStackSys,
				MeticNameSys,
				MeticNameTotalAlloc,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetGaugeMetrics(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGaugeMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCounterMetrics(t *testing.T) {
	tests := []struct {
		name string
		want []MetricName
	}{
		{
			name: "get counter metrics",
			want: []MetricName{
				MetricNamePollCount,
				MetricNameRandomValue,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCounterMetrics(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCounterMetrics() = %v, want %v", got, tt.want)
			}
		})
	}
}
