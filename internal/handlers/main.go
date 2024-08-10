package handlers

import (
	"io"
	"net/http"
	"strconv"

	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

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

func (h APIHandler) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.Index)
	r.Get("/value/{type}/{name}", h.Value)
	r.Post("/update/{type}/{name}/{value}", h.Update)
	return r
}

func (h APIHandler) Index(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := "<html><body><dl>"
	for name, value := range h.Storage.GetAll() {
		html += "<dt>" + string(name) + "</dt><dd>" + value.String() + "</dd>"
	}
	html += "</dl></body></html>"
	io.WriteString(res, "List"+html)
}

func (h APIHandler) Value(res http.ResponseWriter, req *http.Request) {
	metricName := chi.URLParam(req, "name")
	if metricName == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	m := h.Storage.Get(storage.MetricName(metricName))
	if m == nil {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(res, m.String())
}

func (h APIHandler) Update(res http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {
		http.Error(res, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	contentType := req.Header.Values("Content-Type")
	if len(contentType) == 0 || contentType[0] != "text/plain" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	metricType := chi.URLParam(req, "type")
	if metricType != string(storage.MetricTypeGauge) && metricType != string(storage.MetricTypeCounter) {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	sMetricType := storage.MetricType(metricType)

	metricName := chi.URLParam(req, "name")
	if metricName == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	sMetricName := storage.MetricName(metricName)

	metricValue := chi.URLParam(req, "value")
	if metricValue == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var sValue interface{}

	switch sMetricType {
	case storage.MetricTypeGauge:
		if v, err := strconv.ParseFloat(metricValue, 64); err == nil {
			sValue = v
		} else {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	case storage.MetricTypeCounter:
		if v, err := strconv.ParseInt(metricValue, 10, 64); err == nil {
			sValue = v
		} else {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	if exists := h.Storage.Get(sMetricName); exists != nil {
		h.Storage.Update(sMetricName, sValue)
	} else {
		switch sMetricType {
		case storage.MetricTypeGauge:
			h.Storage.Insert(sMetricName, storage.NewMetricGauge(sValue))
		case storage.MetricTypeCounter:
			h.Storage.Insert(sMetricName, storage.NewMetricCounter(sValue))
		}
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}
