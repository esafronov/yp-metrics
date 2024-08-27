package storage

type Repositories interface {
	Get(MetricName) Metric
	Insert(MetricName, Metric)
	Update(Metric, interface{})
	GetAll() map[MetricName]Metric
}
