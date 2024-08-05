package storage

type Repositories interface {
	Get(MetricName) interface{}
	Insert(MetricName, interface{})
}
