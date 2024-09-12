package storage

import "context"

type Repositories interface {
	//get entry
	Get(context.Context, MetricName) (Metric, error)
	//insert entry
	Insert(context.Context, MetricName, Metric) error
	//update entry
	Update(context.Context, MetricName, interface{}, Metric) error
	//get all entries
	GetAll(context.Context) (map[MetricName]Metric, error)
	//close
	Close(context.Context) error
}
