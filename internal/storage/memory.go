package storage

import "context"

type MemStorage struct {
	Values map[MetricName]Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{Values: make(map[MetricName]Metric)}
}

func (s *MemStorage) Get(ctx context.Context, key MetricName) (Metric, error) {
	if val, ok := s.Values[key]; ok {
		return val, nil
	}
	return nil, nil
}

func (s *MemStorage) Insert(ctx context.Context, key MetricName, m Metric) error {
	s.Values[key] = m
	return nil
}

func (s *MemStorage) Update(ctx context.Context, key MetricName, v interface{}, metric Metric) error {
	s.Values[key].UpdateValue(v)
	return nil
}

func (s *MemStorage) GetAll(ctx context.Context) (map[MetricName]Metric, error) {
	return s.Values, nil
}

func (s *MemStorage) Close(ctx context.Context) error {
	return nil
}
