package storage

type Repositories interface {
	Get(MetricName) Metric
	Insert(MetricName, Metric)
}
