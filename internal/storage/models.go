package storage

type MetricType string
type MetricName string

const (
	MetricTypeGauge   MetricType = "gauge"
	MetricTypeCounter MetricType = "counter"
)

type MetricGauge struct {
	val float64
}

type MetricCounter struct {
	val int64
}

func NewMetricGauge(val float64) *MetricGauge {
	return &MetricGauge{val: val}
}

func NewMetricCounter(val int64) *MetricCounter {
	return &MetricCounter{val: val}
}

func (m *MetricGauge) UpdateValue(v float64) {
	m.val = v
}

func (m *MetricCounter) IncrementValue(v int64) {
	m.val += v
}
