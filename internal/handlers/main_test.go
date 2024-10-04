package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestAPIHandler_UpdateJSON(t *testing.T) {

	type want struct {
		contentType string
		statusCode  int
		metric      storage.Metric
		metricname  storage.MetricName
		body        string
	}

	type request struct {
		path string
		body string
	}

	tests := []struct {
		name    string
		storage *storage.MemStorage
		request *request
		want    want
	}{
		{
			name: "positive update gauge sequence",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricGauge(float64(1.2)),
				},
			},
			request: &request{
				path: "/update/",
				body: `{
					"id":"test",
					"type":"gauge",
					"value":1.1
				}`,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricGauge(float64(1.1)),
				metricname:  storage.MetricName("test"),
				body: `{
					"id":"test",
					"type":"gauge",
					"value":1.1
				}`,
			},
		},
		{
			name: "positive update counter sequence",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricCounter(int64(2)),
				},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"test",
					"type":"counter",
					"delta":2
				}`,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricCounter(int64(4)),
				metricname:  storage.MetricName("test"),
				body: `{
					"id":"test",
					"type":"counter",
					"delta":4
				}`,
			},
		},
		{
			name: "positive new gauge",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"test",
					"type":"gauge",
					"value":1.1
				}`,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricGauge(float64(1.1)),
				metricname:  storage.MetricName("test"),
				body: `{
					"id":"test",
					"type":"gauge",
					"value":1.1
				}`,
			},
		},
		{
			name: "positive new counter",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"test",
					"type":"counter",
					"delta":2
				}`,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
				body: `{
					"id":"test",
					"type":"counter",
					"delta":2
				}`,
			},
		},
		{
			name: "wrong gauge value",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"test",
					"type":"gauge",
					"value":"f"
				}`,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "wrong counter value",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"test",
					"type":"counter",
					"delta":1.1
				}`,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "empty metric name",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"",
					"type":"counter",
					"value":1
				}`,
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name: "wrong metric type",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"test",
					"type":"wrong",
					"value":1
				}`,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "wrong path #1",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/updat",
				body: `{}`,
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAPIHandler(tt.storage, "")
			ts := httptest.NewServer(h.GetRouter())

			defer ts.Close()
			reader := strings.NewReader(tt.request.body)
			req, err := http.NewRequest(http.MethodPost, ts.URL+tt.request.path, reader)
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			result, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer result.Body.Close()
			require.Equal(t, tt.want.statusCode, result.StatusCode)

			if tt.want.statusCode == http.StatusOK {
				require.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))
				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				require.JSONEq(t, tt.want.body, string(body))
				m, err := h.Storage.Get(context.Background(), tt.want.metricname)
				require.NoError(t, err, "ошибка получения метрики")
				require.NotNil(t, m, "отправленная метрика не найдена в хранилище")
				require.Equal(t, tt.want.metric, m, "метрика в хранилище не соответствует ожидаемой")
			}
		})
	}
}

func TestAPIHandler_Updates(t *testing.T) {

	type want struct {
		contentType string
		statusCode  int
		metric      storage.Metric
		metricname  storage.MetricName
		secret      string
	}

	type request struct {
		path   string
		body   string
		secret string
	}

	tests := []struct {
		name    string
		storage *storage.MemStorage
		request *request
		want    want
	}{
		{
			name: "batch positive signature",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/updates/",
				body: `[{
					"ID":"some",
					"type":"counter",
					"delta":1
				}]`,
				secret: "123",
			},
			want: want{
				secret:      "123",
				statusCode:  http.StatusOK,
				contentType: "application/json",
				metric:      storage.NewMetricCounter(int64(1)),
				metricname:  storage.MetricName("some"),
			},
		},
		{
			name: "batch wrong signature",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/updates/",
				body: `[{
					"ID":"some",
					"type":"counter",
					"delta":1
				}]`,
				secret: "wrong_secret",
			},
			want: want{
				secret:     "right_server_secret",
				statusCode: http.StatusBadRequest,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAPIHandler(tt.storage, tt.want.secret)
			ts := httptest.NewServer(h.GetRouter())

			defer ts.Close()
			reader := strings.NewReader(tt.request.body)
			req, err := http.NewRequest(http.MethodPost, ts.URL+tt.request.path, reader)
			require.NoError(t, err)

			req.Header.Set("Content-Type", "application/json")

			//emulate agent has key
			if tt.request.secret != "" {
				signature, err := signing.Sign([]byte(tt.request.body), tt.request.secret)
				require.NoError(t, err, "error signing request for agent")
				req.Header.Set(signing.HeaderSignatureKey, signature)
			}

			result, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer result.Body.Close()
			require.Equal(t, tt.want.statusCode, result.StatusCode)

			if tt.want.statusCode == http.StatusOK {
				require.Equal(t, tt.want.contentType, result.Header.Get("Content-Type"))
				_, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				m, err := h.Storage.Get(context.Background(), tt.want.metricname)
				require.NoError(t, err, "ошибка получения метрики")
				require.NotNil(t, m, "отправленная метрика не найдена в хранилище")
				require.Equal(t, tt.want.metric, m, "метрика в хранилище не соответствует ожидаемой")
			}
		})
	}
}
