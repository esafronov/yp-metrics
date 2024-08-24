package storage

type MemStorage struct {
	Values map[MetricName]Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{Values: make(map[MetricName]Metric)}
}

func (s *MemStorage) Get(key MetricName) Metric {
	if val, ok := s.Values[key]; ok {
		return val
	}
	return nil
}

func (s *MemStorage) Insert(key MetricName, m Metric) {
	s.Values[key] = m
}

func (s *MemStorage) Update(key MetricName, v interface{}) {
	s.Values[key].UpdateValue(v)
}

func (s *MemStorage) GetAll() map[MetricName]Metric {
	return s.Values
}
