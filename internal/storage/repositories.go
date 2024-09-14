package storage

import "context"

type Repositories interface {
	//get one entry
	Get(context.Context, MetricName) (Metric, error)
	//insert one entry
	Insert(context.Context, MetricName, Metric) error
	//update one entry
	Update(context.Context, MetricName, interface{}, Metric) error
	//get all entries
	GetAll(context.Context) (map[MetricName]Metric, error)
	//close repository
	Close(context.Context) error
	//update multiple entries
	BatchUpdate(context.Context, []Metrics) error
}
