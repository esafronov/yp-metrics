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

type ErrorPathFormat struct{}

func (e ErrorPathFormat) Error() string {
	return "url path has wrong format"
}

type ErrorPathValue struct{}

func (e ErrorPathValue) Error() string {
	return "url path has wrong values"
}

func ParseURL(p string) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	chunks := strings.Split(p, "/")

	if len(chunks) != 5 {
		return nil, ErrorPathFormat{}
	}

	if chunks[0] != "" {
		return nil, ErrorPathFormat{}
	}

	if chunks[1] != "update" {
		return nil, ErrorPathFormat{}
	}

	if chunks[2] != "gauge" && chunks[2] != "counter" {
		return nil, ErrorPathValue{}
	}

	params["type"] = storage.MetricType(chunks[2])

	if chunks[3] == "" {
		return nil, ErrorPathFormat{}
	}

	params["name"] = storage.MetricName(chunks[3])

	switch params["type"] {
	case storage.MetricTypeGauge:
		v, err := strconv.ParseFloat(chunks[4], 64)
		if err != nil {
			return nil, ErrorPathValue{}
		}
		params["value"] = v
	case storage.MetricTypeCounter:
		v, err := strconv.ParseInt(chunks[4], 10, 64)
		if err != nil {
			return nil, ErrorPathValue{}
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
	ContentType := req.Header.Values("Content-Type")
	if len(ContentType) == 0 || ContentType[0] != "text/plain" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	params, err := ParseURL(req.URL.Path)

	if err != nil {
		switch err.(type) {
		case ErrorPathValue:
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		case ErrorPathFormat:
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
	}

	MetricName, _ := params["name"].(storage.MetricName)
	MetricType, _ := params["type"].(storage.MetricType)

	metric := storageInstance.Get(MetricName)
	if metric != nil {
		metric.UpdateValue(params["value"])
	} else {
		switch MetricType {
		case storage.MetricTypeGauge:
			value, _ := params["value"].(float64)
			storageInstance.Insert(MetricName, storage.NewMetricGauge(value))
		case storage.MetricTypeCounter:
			value, _ := params["value"].(int64)
			storageInstance.Insert(MetricName, storage.NewMetricCounter(value))
		}
	}
	storageInstance.Print()
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}
