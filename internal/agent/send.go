package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/esafronov/yp-metrics/internal/compress"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/retry"
	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
)

// Read metrics from repository and send them to send channel
func (a *Agent) SendMetrics(ctx context.Context, reportInterval *int, rateLimit *int) {
	if *rateLimit > 0 {
		for i := 1; i <= *rateLimit; i++ {
			go a.sendWorker(ctx, i)
		}
	}
	ticker := time.NewTicker(time.Duration(*reportInterval) * time.Second)
	defer close(a.chSend)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if *rateLimit == 0 {
				if err := a.sendReportInBatch(ctx); err != nil {
					logger.Log.Error(err.Error())
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
						a.chSend <- storage.Metrics{
							ID:          string(metricName),
							ActualValue: v.GetValue(),
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
	if *secretKey != "" {
		signature, err := signing.Sign(encodedData, *secretKey)
		if err != nil {
			return fmt.Errorf("signing request : %w", err)
		}
		req.Header.Set(signing.HEADER_SIGNATURE_KEY, signature)
	}
	//header Accept-Encoding : gzip will be added automatically, so not need to add
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	res, err := retry.DoRequest(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("response status: %d", res.StatusCode)
	}
	return nil
}

// Send worker, receives metrics from send channel and call sendMetric function
func (a *Agent) sendWorker(ctx context.Context, num int) {
	fmt.Printf("send worker #%d started...\r\n", num)
	for metric := range a.chSend {
		err := a.sendMetric(ctx, &metric)
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
	res, err := retry.DoRequest(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("response status: %d", res.StatusCode)
	}
	return nil
}
