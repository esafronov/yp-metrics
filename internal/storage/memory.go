package storage

import (
	"context"
	"fmt"
	"sync"
)

type MemStorage struct {
	Values map[MetricName]Metric
	mu     sync.Mutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{Values: make(map[MetricName]Metric)}
}

func (s *MemStorage) Get(ctx context.Context, key MetricName) (Metric, error) {
	s.mu.Lock()
	if val, ok := s.Values[key]; ok {
		s.mu.Unlock()
		return val, nil
	}
	s.mu.Unlock()
	return nil, nil
}

func (s *MemStorage) Insert(ctx context.Context, key MetricName, m Metric) error {
	s.mu.Lock()
	s.Values[key] = m
	s.mu.Unlock()
	return nil
}

func (s *MemStorage) Update(ctx context.Context, key MetricName, v interface{}, metric Metric) error {
	s.mu.Lock()
	s.Values[key].UpdateValue(v)
	s.mu.Unlock()
	return nil
}

func (s *MemStorage) GetAll(ctx context.Context) (map[MetricName]Metric, error) {
	return s.Values, nil
}

func (s *MemStorage) Close(ctx context.Context) error {
	return nil
}

func (s *MemStorage) BatchUpdate(ctx context.Context, metrics []Metrics) error {
	for _, m := range metrics {
		metric, err := s.Get(ctx, MetricName(m.ID))
		if err != nil {
			return err
		}
		if metric != nil {
			err = s.Update(ctx, MetricName(m.ID), m.ActualValue, metric)
		} else {
			switch val := m.ActualValue.(type) {
			case int64:
				err = s.Insert(ctx, MetricName(m.ID), NewMetricCounter(val))
			case float64:
				err = s.Insert(ctx, MetricName(m.ID), NewMetricGauge(val))
			default:
				err = fmt.Errorf("metric type unknown in batch update")
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
