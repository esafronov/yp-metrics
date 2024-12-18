package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/stretchr/testify/require"
)

// benchmark for updating metrica with JSON
func BenchmarkHandler_UpdateJSON(b *testing.B) {
	h := NewAPIHandler(storage.NewMemStorage(), "")
	testN := 1
	testStr := "test" + strconv.Itoa(testN)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		reader := strings.NewReader(`
			{
				"id":"` + testStr + `",
				"type":"counter",
				"delta":1
			}
		`)
		req := httptest.NewRequest("POST", "/update/", reader)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		b.StartTimer()
		h.UpdateJSON(w, req)
		response := w.Result()
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			b.Error("wrong response status", response.StatusCode)
		}
		if testN%10 == 0 {
			testN++
			testStr = "test" + strconv.Itoa(testN)
		}
	}
}

func TestAPIHandler_Update(t *testing.T) {

	type want struct {
		contentType string
		statusCode  int
		metric      storage.Metric
		metricname  storage.MetricName
	}

	type request struct {
		path string
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
				path: "/update/gauge/test/1.1",
			},
			want: want{
				statusCode: http.StatusOK,
				metric:     storage.NewMetricGauge(float64(1.1)),
				metricname: storage.MetricName("test"),
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
				path: "/update/counter/test/2",
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricCounter(int64(4)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "positive new gauge",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/gauge/test/1.1",
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricGauge(float64(1.1)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "positive new counter",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/counter/test/2",
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
			},
		},
		{
			name: "wrong gauge value",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/gauge/test/f",
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
				path: "/update/counter/test/1.1",
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
				path: "/update/counter//1",
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
				path: "/update//test/1",
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
			req, err := http.NewRequest(http.MethodPost, ts.URL+tt.request.path, nil)
			require.NoError(t, err)

			result, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer result.Body.Close()
			require.Equal(t, tt.want.statusCode, result.StatusCode)

			if tt.want.statusCode == http.StatusOK {
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
		path        string
		body        string
		secret      string
		contentType string
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
		{
			name: "wrong media content type",
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
				secret:      "123",
				contentType: "html/text",
			},
			want: want{
				secret:     "123",
				statusCode: http.StatusUnsupportedMediaType,
			},
		},
		{
			name: "wrong metric type",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/updates/",
				body: `[{
					"ID":"some",
					"type":"wrongtype",
					"delta":1
				}]`,
				secret: "123",
			},
			want: want{
				secret:      "123",
				statusCode:  http.StatusBadRequest,
				contentType: "html/text",
			},
		},
		{
			name: "wrong metric name",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/updates/",
				body: `[{
					"ID":"",
					"type":"counter",
					"delta":1
				}]`,
				secret: "123",
			},
			want: want{
				secret:      "123",
				statusCode:  http.StatusNotFound,
				contentType: "html/text",
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

			//if content type is set we use it for test
			if tt.request.contentType != "" {
				req.Header.Set("Content-Type", tt.request.contentType)
			} else {
				req.Header.Set("Content-Type", "application/json")
			}

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

func ExampleAPIHandler_Index() {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer req.Body.Close()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}

func ExampleAPIHandler_Ping() {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/ping/", nil)
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer req.Body.Close()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}

func ExampleAPIHandler_Update() {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/counter/test/1", nil)
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer req.Body.Close()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}

func ExampleAPIHandler_UpdateJSON() {
	reader := strings.NewReader(`{
		"ID":"test",          
		"MType":"counter",     
		"Delta":1
	}`)
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/update/", reader)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer req.Body.Close()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}

func ExampleAPIHandler_Updates() {
	reader := strings.NewReader(`[{
		"ID":"test",          
		"MType":"counter",     
		"Delta":1
	},{
		"ID":"test2",          
		"MType":"gauge",     
		"Value":0.001
	}]`)
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/updates/", reader)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer req.Body.Close()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}

func ExampleAPIHandler_Value() {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/value/counter/test", nil)
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer req.Body.Close()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}

func ExampleAPIHandler_ValueJSON() {
	reader := strings.NewReader(`{
		"ID":"test",
		"MType":"counter"
	}`)
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/value/", reader)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer req.Body.Close()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}
