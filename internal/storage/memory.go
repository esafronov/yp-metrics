package storage

import "fmt"

type MemStorage struct {
	values map[MetricName]Metric
}

func NewMemStorage() *MemStorage {
	return &MemStorage{values: make(map[MetricName]Metric)}
}

func (storage *MemStorage) Get(key MetricName) Metric {
	if val, ok := storage.values[key]; ok {
		return val
	}
	return nil
}

func (storage *MemStorage) Insert(key MetricName, m Metric) {
	storage.values[key] = m
}

func (storage *MemStorage) Print() {
	for n, m := range storage.values {
		fmt.Printf("n: %s, m: %v", n, m)
	}
}
