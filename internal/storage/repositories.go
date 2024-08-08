package storage

type Repositories interface {
	Get(MetricName) Metric
	Insert(MetricName, Metric)
	Update(MetricName, interface{})
	GetAll() map[MetricName]Metric
}
