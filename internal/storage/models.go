package storage

import "fmt"

type MetricType string
type MetricName string

const (
	MetricNameAlloc        MetricName = "Alloc"
	MetricNameBuckHashSys  MetricName = "BuckHashSys"
	MetricNameBuckFrees    MetricName = "Frees"
	MeticNameGCCPUFraction MetricName = "GCCPUFraction"
	MeticNameGCSys         MetricName = "GCSys"
	MeticNameHeapAlloc     MetricName = "HeapAlloc"
	MeticNameHeapIdle      MetricName = "HeapIdle"
	MeticNameHeapInuse     MetricName = "HeapInuse"
	MeticNameHeapObjects   MetricName = "HeapObjects"
	MeticNameHeapReleased  MetricName = "HeapReleased"
	MeticNameHeapSys       MetricName = "HeapSys"
	MeticNameLastGC        MetricName = "LastGC"
	MeticNameLookups       MetricName = "Lookups"
	MeticNameMCacheInuse   MetricName = "MCacheInuse"
	MeticNameMCacheSys     MetricName = "MCacheSys"
	MeticNameMSpanInuse    MetricName = "MSpanInuse"
	MeticNameMSpanSys      MetricName = "MSpanSys"
	MeticNameMallocs       MetricName = "Mallocs"
	MeticNameNextGC        MetricName = "NextGC"
	MeticNameNumForcedGC   MetricName = "NumForcedGC"
	MeticNameNumGC         MetricName = "NumGC"
	MeticNameOtherSys      MetricName = "OtherSys"
	MeticNamePauseTotalNs  MetricName = "PauseTotalNs"
	MeticNameStackInuse    MetricName = "StackInuse"
	MeticNameStackSys      MetricName = "StackSys"
	MeticNameSys           MetricName = "Sys"
	MeticNameTotalAlloc    MetricName = "TotalAlloc"
	MetricNamePollCount    MetricName = "PollCount"
	MetricNameRandomValue  MetricName = "RandomValue"
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
	return fmt.Sprintf("%f", m.val)
}

func (m *MetricCounter) String() string {
	return fmt.Sprint(m.val)
}
