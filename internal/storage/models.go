// Package storage includes repository implementations: MemStorage, HybridStorage, DbStorage, DTO objects and entities
package storage

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type MetricType string
type MetricName string

const (
	MetricNameAlloc           MetricName = "Alloc"
	MetricNameBuckHashSys     MetricName = "BuckHashSys"
	MetricNameBuckFrees       MetricName = "Frees"
	MeticNameGCCPUFraction    MetricName = "GCCPUFraction"
	MeticNameGCSys            MetricName = "GCSys"
	MeticNameHeapAlloc        MetricName = "HeapAlloc"
	MeticNameHeapIdle         MetricName = "HeapIdle"
	MeticNameHeapInuse        MetricName = "HeapInuse"
	MeticNameHeapObjects      MetricName = "HeapObjects"
	MeticNameHeapReleased     MetricName = "HeapReleased"
	MeticNameHeapSys          MetricName = "HeapSys"
	MeticNameLastGC           MetricName = "LastGC"
	MeticNameLookups          MetricName = "Lookups"
	MeticNameMCacheInuse      MetricName = "MCacheInuse"
	MeticNameMCacheSys        MetricName = "MCacheSys"
	MeticNameMSpanInuse       MetricName = "MSpanInuse"
	MeticNameMSpanSys         MetricName = "MSpanSys"
	MeticNameMallocs          MetricName = "Mallocs"
	MeticNameNextGC           MetricName = "NextGC"
	MeticNameNumForcedGC      MetricName = "NumForcedGC"
	MeticNameNumGC            MetricName = "NumGC"
	MeticNameOtherSys         MetricName = "OtherSys"
	MeticNamePauseTotalNs     MetricName = "PauseTotalNs"
	MeticNameStackInuse       MetricName = "StackInuse"
	MeticNameStackSys         MetricName = "StackSys"
	MeticNameSys              MetricName = "Sys"
	MeticNameTotalAlloc       MetricName = "TotalAlloc"
	MetricNamePollCount       MetricName = "PollCount"
	MetricNameRandomValue     MetricName = "RandomValue"
	MetricNameTotalMemory     MetricName = "TotalMemory"
	MetricNameFreeMemory      MetricName = "FreeMemory"
	MetricNameCPUutilization1 MetricName = "CPUutilization1"
)

func GetGaugeMetrics() []MetricName {
	return []MetricName{
		MetricNameAlloc,
		MetricNameBuckHashSys,
		MetricNameBuckFrees,
		MeticNameGCCPUFraction,
		MeticNameGCSys,
		MeticNameHeapAlloc,
		MeticNameHeapIdle,
		MeticNameHeapInuse,
		MeticNameHeapObjects,
		MeticNameHeapReleased,
		MeticNameHeapSys,
		MeticNameLastGC,
		MeticNameLookups,
		MeticNameMCacheInuse,
		MeticNameMCacheSys,
		MeticNameMSpanInuse,
		MeticNameMSpanSys,
		MeticNameMallocs,
		MeticNameNextGC,
		MeticNameNumForcedGC,
		MeticNameNumGC,
		MeticNameOtherSys,
		MeticNamePauseTotalNs,
		MeticNameStackInuse,
		MeticNameStackSys,
		MeticNameSys,
		MeticNameTotalAlloc,
	}
}

func GetCounterMetrics() []MetricName {
	return []MetricName{
		MetricNamePollCount,
		MetricNameRandomValue,
	}
}

const (
	MetricTypeGauge   MetricType = "gauge"
	MetricTypeCounter MetricType = "counter"
)

type Metric interface {
	GetValue() interface{}
	UpdateValue(interface{})
	String() string
}

type MetricGauge struct {
	val float64
}

type MetricCounter struct {
	val int64
}

func NewMetricGauge(val interface{}) Metric {
	return &MetricGauge{val: val.(float64)}
}

func NewMetricCounter(val interface{}) Metric {
	return &MetricCounter{val: val.(int64)}
}

func (m *MetricGauge) UpdateValue(v interface{}) {
	m.val = v.(float64)
}

func (m *MetricCounter) UpdateValue(v interface{}) {
	m.val += v.(int64)
}

func (m *MetricGauge) GetValue() interface{} {
	return m.val
}

func (m *MetricCounter) GetValue() interface{} {
	return m.val
}

func (m *MetricGauge) String() string {
	return strconv.FormatFloat(m.val, 'f', -1, 64)
}

func (m *MetricCounter) String() string {
	return strconv.FormatInt(m.val, 10)
}

// Metrics is DTO
type Metrics struct {
	ActualValue interface{} `json:"-"`
	Value       *float64    `json:"value,omitempty"`
	Delta       *int64      `json:"delta,omitempty"`
	ID          string      `json:"id"`
	MType       string      `json:"type"`
}

func (m *Metrics) UnmarshalJSON(data []byte) (err error) {
	type MetricsAlias Metrics

	aliasValue := &struct {
		*MetricsAlias
	}{
		MetricsAlias: (*MetricsAlias)(m),
	}
	if err := json.Unmarshal(data, aliasValue); err != nil {
		return err
	}
	switch MetricType(m.MType) {
	case MetricTypeGauge:
		if m.Value != nil {
			m.ActualValue = *m.Value
		}
	case MetricTypeCounter:
		if m.Delta != nil {
			m.ActualValue = *m.Delta
		}
	default:
		return fmt.Errorf("wrong metric type %s", m.MType)
	}
	return
}

func (m Metrics) MarshalJSON() ([]byte, error) {
	// чтобы избежать рекурсии при json.Marshal, объявляем новый тип
	/*type MetricsAlias struct {
		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
		Metrics
	}*/
	type MetricsAlias Metrics
	//	var aliasValue Metrics
	if m.ActualValue == nil {
		return nil, fmt.Errorf("ActualValue is nil")
	}
	switch m.ActualValue.(type) {
	case float64:
		value := m.ActualValue.(float64)
		m.MType = string(MetricTypeGauge)
		aliasValue := struct {
			Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
			Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
			MetricsAlias
		}{
			MetricsAlias: MetricsAlias(m),
			Value:        &value,
		}
		return json.Marshal(aliasValue)
	case int64:
		delta := m.ActualValue.(int64)
		m.MType = string(MetricTypeCounter)
		aliasValue := struct {
			Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
			Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
			MetricsAlias
		}{
			MetricsAlias: MetricsAlias(m),
			Delta:        &delta,
		}
		return json.Marshal(aliasValue)
	default:
		return nil, fmt.Errorf("wrong metric type")
	}

}
