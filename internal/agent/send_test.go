package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/esafronov/yp-metrics/internal/compress"
	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/stretchr/testify/require"
)

func JSONArraysEq(t *testing.T, expectedJSON string, actualJSON string, msgAndArgs ...interface{}) {
	var expected, actual interface{}
	require.NoError(t, json.Unmarshal([]byte(expectedJSON), &expected))
	require.NoError(t, json.Unmarshal([]byte(actualJSON), &actual))
	require.ElementsMatch(t, expected, actual, msgAndArgs...)
}

func TestAgent_SendMetrics(t *testing.T) {
	type request struct {
		path string
		body string
	}

	type want struct {
		request     *request
		contentType string
	}

	tests := []struct {
		want           want
		a              *Agent
		name           string
		secretKey      string
		reportInterval int
		rateLimit      int
	}{
		{
			name: "send gauge Lookups 1.200000",
			a: &Agent{
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"Lookups": storage.NewMetricGauge(float64(1.200000)),
					},
				},
				chSend: make(chan storage.Metrics),
			},
			reportInterval: 1,
			rateLimit:      1,
			want: want{
				contentType: "application/json",
				request: &request{
					path: "/update/",
					body: `{
						"id" : "Lookups",
						"type" : "gauge",
						"value" : 1.200000
					}`,
				},
			},
		},
		{
			name: "send counter PollCount 1",
			a: &Agent{
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"PollCount": storage.NewMetricCounter(int64(1)),
					},
				},
				chSend: make(chan storage.Metrics),
			},
			reportInterval: 1,
			rateLimit:      1,
			want: want{
				contentType: "application/json",
				request: &request{
					path: "/update/",
					body: `{
						"id" : "PollCount",
						"type" : "counter",
						"delta" : 1
					}`,
				},
			},
		},
		{
			name:      "batch send gauge metrics",
			secretKey: "mypass",
			a: &Agent{
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"Lookups": storage.NewMetricGauge(float64(1.200000)),
						"Test":    storage.NewMetricGauge(float64(1.0002)),
					},
				},
				chSend:    make(chan storage.Metrics),
				secretKey: "mypass",
			},
			reportInterval: 1,
			rateLimit:      0,
			want: want{
				contentType: "application/json",
				request: &request{
					path: "/updates/",
					body: `[{
						"id" : "Lookups",
						"type" : "gauge",
						"value" : 1.200000
					},{
						"id" : "Test",
						"type" : "gauge",
						"value" : 1.0002
					}]`,
				},
			},
		},
		{
			name:      "batch send counter metrics",
			secretKey: "",
			a: &Agent{
				storage: &storage.MemStorage{
					Values: map[storage.MetricName]storage.Metric{
						"PollCount": storage.NewMetricCounter(int64(1)),
						"testCount": storage.NewMetricCounter(int64(2)),
					},
				},
				chSend: make(chan storage.Metrics),
			},
			reportInterval: 1,
			rateLimit:      0,
			want: want{
				contentType: "application/json",
				request: &request{
					path: "/updates/",
					body: `[{
						"id" : "PollCount",
						"type" : "counter",
						"delta" : 1
					},{
						"id" : "testCount",
						"type" : "counter",
						"delta" : 2
					}]`,
				},
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(compress.GzipCompressing(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				require.Equal(t, tt.want.contentType, req.Header.Get("Content-Type"))
				require.Equal(t, tt.want.request.path, req.URL.String())
				body, err := io.ReadAll(req.Body)
				require.Nil(t, err, "error reading request body")
				//fmt.Println(string(body))
				//if secretKey is not empty then we should get signature from agent and check it is valid
				if tt.secretKey != "" {
					signature := req.Header.Get(signing.HeaderSignatureKey)
					require.NotEmpty(t, signature, "signature should not be empty")
					isValid := signing.IsValid(signature, body, tt.secretKey)
					require.True(t, isValid, "signature is not valid")
				}
				if tt.rateLimit > 0 {
					//single metric
					require.JSONEq(t, tt.want.request.body, string(body))
				} else {
					//batch metrics
					JSONArraysEq(t, tt.want.request.body, string(body))
				}
			})))
			// Close the server when test finishes
			defer server.Close()
			tt.a.serverAddress = server.URL
			ctx, cancel := context.WithCancel(context.Background())
			go tt.a.SendMetrics(ctx, &tt.reportInterval, &tt.rateLimit)
			time.Sleep(time.Duration(1500) * time.Millisecond)
			cancel()
		})
	}
}
