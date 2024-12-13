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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/esafronov/yp-metrics/internal/pg"
	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/stretchr/testify/assert"
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
		defer func() {
			err := response.Body.Close()
			if err != nil {
				b.Error("body close", err)
			}
		}()
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
		metric      storage.Metric
		contentType string
		metricname  storage.MetricName
		statusCode  int
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
			defer func() {
				err := result.Body.Close()
				if err != nil {
					assert.NoError(t, err)
				}
			}()
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
		metric      storage.Metric
		contentType string
		metricname  storage.MetricName
		body        string
		statusCode  int
	}

	type request struct {
		path        string
		body        string
		contentType string
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
		{
			name: "wrong content type",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				contentType: "text/html",
				path:        "/update/",
				body:        `{}`,
			},
			want: want{
				statusCode: http.StatusUnsupportedMediaType,
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

			if tt.request.contentType != "" {
				req.Header.Set("Content-Type", tt.request.contentType)
			} else {
				req.Header.Set("Content-Type", "application/json")
			}

			result, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer func() {
				err := result.Body.Close()
				if err != nil {
					assert.NoError(t, err)
				}
			}()
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
		metric      storage.Metric
		contentType string
		metricname  storage.MetricName
		secret      string
		statusCode  int
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
				var signature string
				signature, err = signing.Sign([]byte(tt.request.body), tt.request.secret)
				require.NoError(t, err, "error signing request for agent")
				req.Header.Set(signing.HeaderSignatureKey, signature)
			}

			result, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer func() {
				err := result.Body.Close()
				if err != nil {
					assert.NoError(t, err)
				}
			}()
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
	defer func() {
		err = req.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
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
	defer func() {
		err = req.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
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
	defer func() {
		err = req.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
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
	if req == nil {
		fmt.Printf("request is nil")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer func() {
		err = req.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
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
	if req == nil {
		fmt.Printf("request is nil")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer func() {
		err = req.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
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
	if req == nil {
		fmt.Printf("request is nil")
		return
	}
	defer func() {
		err = req.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
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
	if req == nil {
		fmt.Printf("request is nil")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		fmt.Println("new request error:", err)
		return
	}
	defer func() {
		err = req.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("do request error:", err)
		return
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			fmt.Println("body close error:", err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
	if res.StatusCode != http.StatusOK {
		fmt.Printf("response status: %d", res.StatusCode)
	}
}

func TestAPIHandler_Index(t *testing.T) {

	s := &storage.MemStorage{
		Values: map[storage.MetricName]storage.Metric{
			"test": storage.NewMetricGauge(float64(1.2)),
		},
	}

	h := NewAPIHandler(s, "")
	ts := httptest.NewServer(h.GetRouter())

	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	require.NoError(t, err)
	if req == nil {
		fmt.Printf("request is nil")
		return
	}
	res, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		err := res.Body.Close()
		if err != nil {
			assert.NoError(t, err)
		}
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
	require.Equal(t, http.StatusOK, res.StatusCode)
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
	if res.StatusCode == http.StatusOK {
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(
			t,
			`<html><body><table border="1"><tr><td>test</td><td>1.2</td></tr></table></body></html>`,
			string(body),
			"response is not match wanted",
		)
	}
}

func TestAPIHandler_ValueJSON(t *testing.T) {

	type want struct {
		metric      storage.Metric
		contentType string
		metricname  storage.MetricName
		body        string
		statusCode  int
	}

	type request struct {
		path        string
		body        string
		contentType string
	}

	tests := []struct {
		name    string
		storage *storage.MemStorage
		request *request
		want    want
	}{
		{
			name: "positive get gauge metric",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricGauge(float64(1.2)),
				},
			},
			request: &request{
				path: "/value/",
				body: `{
					"id":"test",
					"type":"gauge"
				}`,
			},
			want: want{
				contentType: "application/json",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricGauge(float64(1.2)),
				metricname:  storage.MetricName("test"),
				body: `{
					"id":"test",
					"type":"gauge",
					"value":1.2
				}`,
			},
		},
		{
			name: "positive get counter metric",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricCounter(int64(2)),
				},
			},
			request: &request{
				path: "/value/",
				body: `{
					"id":"test",
					"type":"counter"
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
			name: "metric not found by name",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/value/",
				body: `{
					"ID":"unknown",
					"type":"counter"
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
				path: "/value/",
				body: `{
					"ID":"test",
					"type":"wrong"
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
					"type":"counter"
				}`,
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name: "wrong json request",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/update/",
				body: `{
					"ID":"",
					"type":
				}`,
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name: "wrong request content type",

			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				contentType: "text/html",
				path:        "/update/",
				body: `{
					"ID":"test",
					"type": "counter"
				}`,
			},
			want: want{
				statusCode: http.StatusUnsupportedMediaType,
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
			if tt.request.contentType != "" {
				req.Header.Set("Content-Type", tt.request.contentType)
			} else {
				req.Header.Set("Content-Type", "application/json")
			}
			res, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer func() {
				err := res.Body.Close()
				assert.NoError(t, err)
			}()
			require.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.statusCode == http.StatusOK {
				require.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
				body, err := io.ReadAll(res.Body)
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

func TestAPIHandler_Ping(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	mock.ExpectPing()
	s := &storage.DBStorage{}
	require.NoError(t, err)
	pg.DB = db
	h := NewAPIHandler(s, "")
	ts := httptest.NewServer(h.GetRouter())
	defer ts.Close()
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
	require.NoError(t, err)
	res, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		err := res.Body.Close()
		assert.NoError(t, err)
	}()
	if res == nil {
		fmt.Printf("response is nil")
		return
	}
	require.Equal(t, http.StatusOK, res.StatusCode)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func TestAPIHandler_Value(t *testing.T) {
	type want struct {
		metric      storage.Metric
		contentType string
		metricname  storage.MetricName
		body        string
		statusCode  int
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
			name: "positive view gauge value",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricGauge(float64(1.2)),
				},
			},
			request: &request{
				path: "/value/gauge/test",
			},
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricGauge(float64(1.2)),
				metricname:  storage.MetricName("test"),
				body:        "1.2",
			},
		},
		{
			name: "positive view counter value",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{
					"test": storage.NewMetricCounter(int64(2)),
				},
			},
			request: &request{
				path: "/value/counter/test",
			},
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusOK,
				metric:      storage.NewMetricCounter(int64(2)),
				metricname:  storage.MetricName("test"),
				body:        "2",
			},
		},
		{
			name: "empty metric name",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/value/counter/",
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
		},
		{
			name: "metric not found",
			storage: &storage.MemStorage{
				Values: map[storage.MetricName]storage.Metric{},
			},
			request: &request{
				path: "/value/counter/test",
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
			req, err := http.NewRequest(http.MethodGet, ts.URL+tt.request.path, nil)
			require.NoError(t, err)

			result, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer func() {
				err := result.Body.Close()
				assert.NoError(t, err)
			}()
			require.Equal(t, tt.want.statusCode, result.StatusCode)

			if tt.want.statusCode == http.StatusOK {
				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				require.Equal(t, tt.want.body, string(body), "ответ не соответствуюет ожиданию")
				m, err := h.Storage.Get(context.Background(), tt.want.metricname)
				require.NoError(t, err, "ошибка получения метрики")
				require.NotNil(t, m, "отправленная метрика не найдена в хранилище")
				require.Equal(t, tt.want.metric, m, "метрика в хранилище не соответствует ожидаемой")
			}
		})
	}
}
