package storage

type MemStorage struct {
	values map[MetricName]interface{}
}

func NewMemStorage() *MemStorage {
	return &MemStorage{values: make(map[MetricName]interface{})}
}

func (storage *MemStorage) Get(key MetricName) interface{} {
	if val, ok := storage.values[key]; ok {
		return val
	}
	return nil
}

func (storage *MemStorage) Insert(key MetricName, m interface{}) {
	storage.values[key] = m
}
