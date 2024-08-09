package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/esafronov/yp-metrics/internal/storage"
)

type ErrorPathFormat struct{}

func (e ErrorPathFormat) Error() string {
	return "path has wrong format"
}

type ErrorPathValue struct{}

func (e ErrorPathValue) Error() string {
	return "path has wrong values"
}

func ParseUpdatePATH(p string) (*UpdateParams, error) {
	params := &UpdateParams{}

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

	if chunks[2] != string(storage.MetricTypeGauge) && chunks[2] != string(storage.MetricTypeCounter) {
		return nil, ErrorPathValue{}
	}

	params.MetricType = storage.MetricType(chunks[2])

	if chunks[3] == "" {
		return nil, ErrorPathFormat{}
	}

	params.MetricName = storage.MetricName(chunks[3])

	switch params.MetricType {
	case storage.MetricTypeGauge:
		v, err := strconv.ParseFloat(chunks[4], 64)
		if err != nil {
			return nil, ErrorPathValue{}
		}
		params.Value = v
	case storage.MetricTypeCounter:
		v, err := strconv.ParseInt(chunks[4], 10, 64)
		if err != nil {
			return nil, ErrorPathValue{}
		}
		params.Value = v
	}
	return params, nil
}

type APIHandler struct {
	Storage storage.Repositories
}

func NewAPIHandler(s storage.Repositories) *APIHandler {
	return &APIHandler{Storage: s}
}

type UpdateParams struct {
	MetricName storage.MetricName
	MetricType storage.MetricType
	Value      interface{}
}

func (h APIHandler) Update(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	ContentType := req.Header.Values("Content-Type")
	if len(ContentType) == 0 || ContentType[0] != "text/plain" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	params, err := ParseUpdatePATH(req.URL.Path)

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

	if exists := h.Storage.Get(params.MetricName); exists != nil {
		h.Storage.Update(params.MetricName, params.Value)
	} else {
		switch params.MetricType {
		case storage.MetricTypeGauge:
			h.Storage.Insert(params.MetricName, storage.NewMetricGauge(params.Value))
		case storage.MetricTypeCounter:
			h.Storage.Insert(params.MetricName, storage.NewMetricCounter(params.Value))
		}
	}
	res.Header().Add("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}
