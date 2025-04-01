package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/esafronov/yp-metrics/internal/compress"
	"github.com/esafronov/yp-metrics/internal/encrypt"
	pb "github.com/esafronov/yp-metrics/internal/grpc/proto"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/retry"
	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
)

// SendMetrics read metrics from repository and send them to send channel
func (a *Agent) SendMetrics(ctx context.Context, reportInterval *int, rateLimit *int) {
	if rateLimit == nil {
		panic("rateLimit is nil")
	}
	if *rateLimit > 0 {
		for i := 1; i <= *rateLimit; i++ {
			go a.sendWorker(ctx, i)
		}
	}
	if reportInterval == nil {
		panic("reportInterval nil")
	}
	ticker := time.NewTicker(time.Duration(*reportInterval) * time.Second)
	defer close(a.chSend)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if *rateLimit == 0 {
				//if metricsClient is defined then send by grpc client
				if a.metricsClient != nil {
					if err := a.sendReportInBatchGRPC(ctx); err != nil {
						logger.Log.Error(err.Error())
					}
				} else {
					if err := a.sendReportInBatch(ctx); err != nil {
						logger.Log.Error(err.Error())
					}
				}
			} else {
				items, err := a.storage.GetAll(ctx)
				if err != nil {
					logger.Log.Error(err.Error())
					return
				}
				for metricName, v := range items {
					select {
					case <-ctx.Done():
						return
					default:
						var mtype string
						switch v.GetValue().(type) {
						case int64:
							mtype = string(storage.MetricTypeCounter)
						case float64:
							mtype = string(storage.MetricTypeGauge)
						default:
							logger.Log.Error("value type assertion is failed, assuming int64 or float64")
							return
						}
						a.chSend <- storage.Metrics{
							ID:          string(metricName),
							ActualValue: v.GetValue(),
							MType:       mtype,
						}
					}
				}
			}
		}
	}
}

// Send metrics in batch from repository
func (a *Agent) sendReportInBatch(ctx context.Context) error {
	var reqMetrics []storage.Metrics

	items, err := a.storage.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("cannot get metrics %w", err)
	}
	for metricName, v := range items {
		reqMetrics = append(reqMetrics, storage.Metrics{
			ID:          string(metricName),
			ActualValue: v.GetValue(),
		})
	}
	encodedData, err := json.Marshal(reqMetrics)
	if err != nil {
		return fmt.Errorf("marshal error %w", err)
	}
	var compressedData bytes.Buffer
	err = compress.GzipToBuffer(encodedData, &compressedData)
	if err != nil {
		return fmt.Errorf("failed compress request %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.serverAddress+"/updates/", &compressedData)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	//if secretKey is not empty we calc hash from request body and send it with header
	if a.secretKey != "" {
		var signature string
		signature, err = signing.Sign(encodedData, a.secretKey)
		if err != nil {
			return fmt.Errorf("signing request : %w", err)
		}
		req.Header.Set(signing.HeaderSignatureKey, signature)
	}
	//header Accept-Encoding : gzip will be added automatically, so not need to add
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	res, err := retry.DoRequest(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			logger.Log.Info(err.Error())
		}
	}()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("response status: %d", res.StatusCode)
	}
	return nil
}

// Send worker, receives metrics from send channel and call sendMetric or sendMetricGRPC function
func (a *Agent) sendWorker(ctx context.Context, num int) {
	fmt.Printf("send worker #%d started...\r\n", num)
	for metric := range a.chSend {
		var err error
		//if we have grpc metricsClient configured, we use it, otherwise we use http client
		if a.metricsClient != nil {
			err = a.sendMetricGRPC(ctx, &metric)
		} else {
			err = a.sendMetric(ctx, &metric)
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				continue
			}
			logger.Log.Error(err.Error())
		}
	}
}

// Send metric to server (internal worker function)
func (a *Agent) sendMetric(ctx context.Context, m *storage.Metrics) error {
	url := a.serverAddress + "/update/"
	marshaled, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal error %w", err)
	}

	if a.cryptoKey != "" {
		marshaled, err = encrypt.EncryptBody(marshaled, a.cryptoKey)
		if err != nil {
			return fmt.Errorf("encrypt error %w", err)
		}
	}

	var data bytes.Buffer
	err = compress.GzipToBuffer(marshaled, &data)
	if err != nil {
		return fmt.Errorf("failed compress request %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &data)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	//header Accept-Encoding : gzip will be added automatically, so not need to add
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	localIp := GetLocalIP()
	//fmt.Println("local IP", localIp)
	//header X-Real-IP with agent ip address
	req.Header.Set("X-Real-IP", localIp)
	res, err := retry.DoRequest(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			logger.Log.Info(err.Error())
		}
	}()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("response status: %d", res.StatusCode)
	}
	return nil
}

// Send metric by grpc client (internal worker function)
func (a *Agent) sendMetricGRPC(ctx context.Context, m *storage.Metrics) error {
	pbMetric := &pb.Metric{
		Id: m.ID,
	}
	switch storage.MetricType(m.MType) {
	case storage.MetricTypeCounter:
		pbMetric.Type = pb.Metric_COUNTER
		value, ok := m.ActualValue.(int64)
		if !ok {
			return fmt.Errorf("error type assertion, assuming int64")
		}
		pbMetric.Delta = value
	case storage.MetricTypeGauge:
		pbMetric.Type = pb.Metric_GAUGE
		value, ok := m.ActualValue.(float64)
		if !ok {
			return fmt.Errorf("error type assertion, assuming float64")
		}
		pbMetric.Value = value
	default:
		return fmt.Errorf("metric type is unknown: %s", m.MType)
	}
	req := &pb.UpdateRequest{
		Metric: pbMetric,
	}
	_, err := a.metricsClient.Update(ctx, req)
	if err != nil {
		return fmt.Errorf("update request failed: %w", err)
	}
	return nil
}

// Send metrics batch by grpc client
func (a *Agent) sendReportInBatchGRPC(ctx context.Context) error {
	in := &pb.BatchUpdateRequest{}
	items, err := a.storage.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("cannot get metrics %w", err)
	}
	for metricName, v := range items {
		pbMetric := &pb.Metric{
			Id: string(metricName),
		}
		switch v.GetValue().(type) {
		case int64:
			pbMetric.Type = pb.Metric_COUNTER
			value, ok := v.GetValue().(int64)
			if !ok {
				return fmt.Errorf("error type assertion, assuming int64")
			}
			pbMetric.Delta = value
		case float64:
			pbMetric.Type = pb.Metric_GAUGE
			value, ok := v.GetValue().(float64)
			if !ok {
				return fmt.Errorf("error type assertion, assuming float64")
			}
			pbMetric.Value = value
		default:
			return fmt.Errorf("unknown metric type")
		}
		in.Metric = append(in.Metric, pbMetric)
	}
	//if secretKey is not empty we calc hash from request and send it with metadata
	if a.secretKey != "" {
		var signature string
		marshaled, err := json.Marshal(in)
		if err != nil {
			return err
		}
		signature, err = signing.Sign(marshaled, a.secretKey)
		if err != nil {
			return fmt.Errorf("signing request : %w", err)
		}
		// later, add some more metadata to the context (e.g. in an interceptor)
		send, _ := metadata.FromOutgoingContext(ctx)
		newMD := metadata.Pairs(signing.HeaderSignatureKey, signature)
		ctx = metadata.NewOutgoingContext(ctx, metadata.Join(send, newMD))
	}
	if _, err := a.metricsClient.BatchUpdate(ctx, in, grpc.UseCompressor(gzip.Name)); err != nil {
		return fmt.Errorf("batch update error %w", err)
	}
	return nil
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
