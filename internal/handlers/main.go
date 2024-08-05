package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/esafronov/yp-metrics/internal/storage"
)

var storageInstance *storage.MemStorage

func init() {
	storageInstance = storage.NewMemStorage()
}

type ErrorStatusNotFound struct{}

func (e ErrorStatusNotFound) Error() string {
	return "name is empty"
}

type ErrorBadRequest struct{}

func (e ErrorBadRequest) Error() string {
	return "bad request"
}

func ParseUrl(p string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	chunks := strings.Split(p, "/")

	if len(chunks) != 5 {
		return nil, ErrorStatusNotFound{}
	}

	if chunks[0] != "" {
		return nil, ErrorStatusNotFound{}
	}

	if chunks[1] != "update" {
		return nil, ErrorStatusNotFound{}
	}

	if chunks[2] != "gauge" && chunks[2] != "counter" {
		return nil, ErrorBadRequest{}
	}

	params["type"] = storage.MetricType(chunks[2])

	if chunks[3] == "" {
		return nil, ErrorStatusNotFound{}
	}

	params["name"] = storage.MetricName(chunks[3])

	switch params["type"] {
	case storage.MetricTypeGauge:
		v, err := strconv.ParseFloat(chunks[4], 64)
		if err != nil {
			return nil, ErrorBadRequest{}
		}
		params["value"] = v
	case storage.MetricTypeCounter:
		v, err := strconv.ParseInt(chunks[4], 10, 64)
		if err != nil {
			return nil, ErrorBadRequest{}
		}
		params["value"] = v
	}
	return params, nil
}

type Main struct{}

func (h Main) ServeHTTP(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	contentType := req.Header.Values("Content-Type")
	if len(contentType) == 0 || contentType[0] != "text/plain" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	params, err := ParseUrl(req.URL.Path)

	if err != nil {
		switch err.(type) {
		case ErrorBadRequest:
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		case ErrorStatusNotFound:
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
	}

	metricName, _ := params["name"].(storage.MetricName)
	metricType, _ := params["type"].(storage.MetricType)

	metric := storageInstance.Get(metricName)

	switch metricType {
	case storage.MetricTypeGauge:
		value, _ := params["value"].(float64)
		metric := storageInstance.Get(metricName)
		if metric != nil {
			m, _ := metric.(*storage.MetricGauge)
			m.UpdateValue(value)
		} else {
			storageInstance.Insert(metricName, storage.NewMetricGauge(value))
		}
	case storage.MetricTypeCounter:
		value, _ := params["value"].(int64)
		if metric != nil {
			m, _ := metric.(*storage.MetricCounter)
			m.IncrementValue(value)
		} else {
			storageInstance.Insert(metricName, storage.NewMetricCounter(value))
		}
	}
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}
