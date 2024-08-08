package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain_ServeHTTP(t *testing.T) {

	type want struct {
		contentType string
		statusCode  int
		metric      storage.Metric
		metricname  storage.MetricName
	}

	tests := []struct {
		name    string
		storage storage.MemStorage
		request string
		want    want
	}{
		{
			name: "positive gauge",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricGauge(float64(1.1)),
				},
			},
			request: "/update/gauge/test/1.1",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricGauge(float64(1.1)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "positive counter",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricCounter(int64(2)),
				},
			},
			request: "/update/counter/test/2",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricCounter(int64(4)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "wrong gauge value",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: "/update/gauge/test/f",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				metric:      storage.NewMetricGauge(float64(2)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "wrong counter value",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: "/update/counter/test/1.1",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "empty name value",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: "/update/counter//1.1",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "wrong metric type",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: "/update/cor/rrr/1",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "wrong path #1",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: "/update/",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "wrong path #2",
			storage: storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: "/update/counter/dd/1/",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Main{
				Storage: &tt.storage,
			}
			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
			request.Header = map[string][]string{
				"Content-Type": {"text/plain"},
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, request)

			result := w.Result()

			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			assert.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))

			if tt.want.statusCode == http.StatusOK {
				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				err = result.Body.Close()
				require.NoError(t, err)
				require.Equal(t, "", string(body))

				m := h.Storage.Get(tt.want.metricname)
				require.NotNil(t, m, "отправленная метрика не найдена в хранилище")
				require.Equal(t, m, tt.want.metric, "метрика в хранилище не соответствует ожидаемой")
			}
		})
	}
}
