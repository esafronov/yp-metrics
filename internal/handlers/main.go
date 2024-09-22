package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/esafronov/yp-metrics/internal/compress"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/pg"
	"github.com/esafronov/yp-metrics/internal/signing"
	"github.com/esafronov/yp-metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type APIHandler struct {
	Storage   storage.Repositories //repository
	secretKey string               //secret key for request signature validation
}

func NewAPIHandler(s storage.Repositories, secretKey string) *APIHandler {
	return &APIHandler{Storage: s, secretKey: secretKey}
}

func (h APIHandler) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(logger.RequestLogger)
	r.Use(compress.GzipCompressing)
	r.Get("/", h.Index)    //html table with all stored metrics
	r.Get("/ping", h.Ping) //test DB connection
	r.Route("/update", func(r chi.Router) {
		r.Post("/", h.UpdateJSON)                  //update metric with json request
		r.Post("/{type}/{name}/{value}", h.Update) //update metric with url request
	})
	r.Route("/value", func(r chi.Router) {
		r.Post("/", h.ValueJSON)         //get metric value with json request
		r.Get("/{type}/{name}", h.Value) //get metric value with url request
	})
	r.Route("/updates", func(r chi.Router) {
		r.Use(signing.ValidateSignature(h.secretKey))
		r.Post("/", h.Updates) //batch updating
	})
	return r
}

func (h APIHandler) Ping(res http.ResponseWriter, req *http.Request) {
	if err := pg.DB.PingContext(req.Context()); err != nil {
		logger.Log.Info("ping error", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)
}

func (h APIHandler) Index(res http.ResponseWriter, req *http.Request) {
	html := `<html><body><table border="1">`
	items, err := h.Storage.GetAll(req.Context())
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	for name, value := range items {
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
	metric, err := h.Storage.Get(req.Context(), metricName)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
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
	metric, err := h.Storage.Get(req.Context(), metricName)
	if err != nil {
		logger.Log.Error("get metric", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if metric != nil {
		err = h.Storage.Update(req.Context(), metricName, value, metric)
		if err != nil {
			logger.Log.Error("update metric", zap.Error(err))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	} else {
		switch metricType {
		case storage.MetricTypeGauge:
			metric = storage.NewMetricGauge(value)
		case storage.MetricTypeCounter:
			metric = storage.NewMetricCounter(value)
		}
		err = h.Storage.Insert(req.Context(), metricName, metric)
		if err != nil {
			logger.Log.Error("insert metric", zap.Error(err))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
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
	metric, err := h.Storage.Get(req.Context(), metricName)
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if metric != nil {
		h.Storage.Update(req.Context(), metricName, value, metric)
	} else {
		switch metricType {
		case storage.MetricTypeGauge:
			h.Storage.Insert(req.Context(), metricName, storage.NewMetricGauge(value))
		case storage.MetricTypeCounter:
			h.Storage.Insert(req.Context(), metricName, storage.NewMetricCounter(value))
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
	m, err := h.Storage.Get(req.Context(), storage.MetricName(mn))
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if m == nil {
		http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(res, m.String())
}

var ErrMetricType = errors.New("metric type is wrong")
var ErrMetricName = errors.New("metric name is empty")

// decode and validate metrics in batch request
func decodeMetrics(body io.ReadCloser) (metrics []storage.Metrics, err error) {
	decoder := json.NewDecoder(body)
	_, err = decoder.Token()
	if err != nil {
		return
	}
	for decoder.More() {
		var m storage.Metrics
		if err = decoder.Decode(&m); err != nil {
			return
		}
		if m.MType != string(storage.MetricTypeGauge) && m.MType != string(storage.MetricTypeCounter) {
			err = ErrMetricType
			return
		}
		if m.ID == "" {
			err = ErrMetricName
			return
		}
		metrics = append(metrics, m)
	}
	_, err = decoder.Token()
	return
}

// batch updating
func (h APIHandler) Updates(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		http.Error(res, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
		return
	}
	metrics, err := decodeMetrics(req.Body)
	if err != nil {
		switch {
		case errors.Is(err, ErrMetricName):
			http.Error(res, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		default:
			http.Error(res, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
		return
	}
	if err := h.Storage.BatchUpdate(req.Context(), metrics); err != nil {
		logger.Log.Error("batch metrics update", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
}
