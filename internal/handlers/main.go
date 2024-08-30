package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/esafronov/yp-metrics/internal/compress"
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
	r.Use(compress.GzipCompressing)
	r.Get("/", h.Index)
	r.Route("/update", func(r chi.Router) {
		r.Post("/", h.UpdateJSON)
		r.Post("/{type}/{name}/{value}", h.Update)
	})
	r.Route("/value", func(r chi.Router) {
		r.Post("/", h.ValueJSON)
		r.Get("/{type}/{name}", h.Value)
	})
	return r
}

func (h APIHandler) Index(res http.ResponseWriter, req *http.Request) {
	html := `<html><body><table border="1">`
	for name, value := range h.Storage.GetAll() {
		html += `<tr><td>` + string(name) + `</td><td>` + value.String() + `</td></tr>`
	}
	html += `</table></body></html>`
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)
	io.WriteString(res, "Storage list:"+html)
}

func (h APIHandler) ValueJSON(res http.ResponseWriter, req *http.Request) {

	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}

	var reqMetric storage.Metrics

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
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	reqMetric.ActualValue = metric.GetValue()
	if err := json.NewEncoder(res).Encode(reqMetric); err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h APIHandler) UpdateJSON(res http.ResponseWriter, req *http.Request) {
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
	value := reqMetric.ActualValue
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
	reqMetric.ActualValue = metric.GetValue()
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(res).Encode(reqMetric); err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h APIHandler) Update(res http.ResponseWriter, req *http.Request) {
	mt := chi.URLParam(req, "type")
	if mt != string(storage.MetricTypeGauge) && mt != string(storage.MetricTypeCounter) {
		http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	metricType := storage.MetricType(mt)

	mn := chi.URLParam(req, "name")
	if mn == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	metricName := storage.MetricName(mn)

	mv := chi.URLParam(req, "value")
	if mv == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var value interface{}

	switch metricType {
	case storage.MetricTypeGauge:
		if v, err := strconv.ParseFloat(mv, 64); err == nil {
			value = v
		} else {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	case storage.MetricTypeCounter:
		if v, err := strconv.ParseInt(mv, 10, 64); err == nil {
			value = v
		} else {
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	if existed := h.Storage.Get(metricName); existed != nil {
		h.Storage.Update(existed, value)
	} else {
		switch metricType {
		case storage.MetricTypeGauge:
			h.Storage.Insert(metricName, storage.NewMetricGauge(value))
		case storage.MetricTypeCounter:
			h.Storage.Insert(metricName, storage.NewMetricCounter(value))
		}
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	res.WriteHeader(http.StatusOK)
}

func (h APIHandler) Value(res http.ResponseWriter, req *http.Request) {
	mn := chi.URLParam(req, "name")
	if mn == "" {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	m := h.Storage.Get(storage.MetricName(mn))
	if m == nil {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(res, m.String())
}
