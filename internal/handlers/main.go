package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
)

type APIHandler struct {
	Storage storage.Repositories
}

func NewAPIHandler(s storage.Repositories) *APIHandler {
	return &APIHandler{Storage: s}
}

func (h APIHandler) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(logger.RequestLogger)
	r.Get("/", h.Index)
	r.Get("/value/", h.Value)
	r.Post("/update/", h.Update)
	return r
}

func (h APIHandler) Index(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<html><body><table border="1">`
	for name, value := range h.Storage.GetAll() {
		html += `<tr><td>` + string(name) + `</td><td>` + value.String() + `</td></tr>`
	}
	html += `</table></body></html>`
	io.WriteString(res, "Storage list:"+html)
}

func (h APIHandler) Value(res http.ResponseWriter, req *http.Request) {

	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	var delta int64
	var value float64
	var reqMetric = storage.Metrics{
		Delta: &delta,
		Value: &value,
	}

	if err := json.NewDecoder(req.Body).Decode(&reqMetric); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if reqMetric.ID == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	metricName := storage.MetricName(reqMetric.ID)

	metric := h.Storage.Get(metricName)
	if metric == nil {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	reqMetric.SetValue(metric)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(reqMetric); err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h APIHandler) Update(res http.ResponseWriter, req *http.Request) {
	var reqMetric storage.Metrics

	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	if err := json.NewDecoder(req.Body).Decode(&reqMetric); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if reqMetric.MType != string(storage.MetricTypeGauge) && reqMetric.MType != string(storage.MetricTypeCounter) {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	metricType := storage.MetricType(reqMetric.MType)
	if reqMetric.ID == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	metricName := storage.MetricName(reqMetric.ID)

	var value interface{}

	switch metricType {
	case storage.MetricTypeGauge:
		value = *reqMetric.Value
	case storage.MetricTypeCounter:
		value = *reqMetric.Delta
	default:
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	metric := h.Storage.Get(metricName)
	if metric != nil {
		h.Storage.Update(metric, value)
	} else {
		switch metricType {
		case storage.MetricTypeGauge:
			metric = storage.NewMetricGauge(value)
		case storage.MetricTypeCounter:
			metric = storage.NewMetricCounter(value)
		}
		h.Storage.Insert(metricName, metric)
	}
	reqMetric.SetValue(metric)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(reqMetric); err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
