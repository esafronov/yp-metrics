package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type HybridStorage struct {
	MemStorage
	file          *os.File      //backup file descriptor
	encoder       *json.Encoder //encoding data->json
	decoder       *json.Decoder //decoding json->data
	storeInterval int           //storeInterval is interval in seconds to make backup
	lastStored    time.Time     //time when backup has been created last time
	backupActive  bool          //is backuping active or not, for internal usage
}

func NewHybridStorage(ctx context.Context, filename string, storeInterval *int, restore *bool) (storage *HybridStorage, err error) {
	var file *os.File
	backupActive := true
	if filename != "" {
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			return
		}
	} else {
		backupActive = false
	}
	encoder := json.NewEncoder(file)
	decoder := json.NewDecoder(file)
	storage = &HybridStorage{
		MemStorage: MemStorage{
			Values: make(map[MetricName]Metric),
		},
		file:          file,
		storeInterval: *storeInterval,
		lastStored:    time.Time{},
		encoder:       encoder,
		decoder:       decoder,
		backupActive:  backupActive,
	}
	if *restore {
		err = storage.Restore(ctx)
	}
	return
}

func (s *HybridStorage) Insert(ctx context.Context, key MetricName, m Metric) error {
	err := s.MemStorage.Insert(ctx, key, m)
	if err != nil {
		return err
	}
	return s.backupCaller(ctx)
}

func (s *HybridStorage) Update(ctx context.Context, key MetricName, v interface{}, metric Metric) error {
	err := s.MemStorage.Update(ctx, key, v, metric)
	if err != nil {
		return err
	}
	return s.backupCaller(ctx)
}

func (s *HybridStorage) backupCaller(ctx context.Context) error {
	if !s.backupActive {
		return nil
	}
	if time.Since(s.lastStored).Seconds() > float64(s.storeInterval) {
		return s.Backup(ctx)
	}
	return nil
}

func (s *HybridStorage) Backup(ctx context.Context) error {
	//truncate file before writing new data
	err := s.file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = s.file.Seek(0, 0)
	if err != nil {
		return err
	}
	s.lastStored = time.Now()
	items, err := s.GetAll(ctx)
	if err != nil {
		return err
	}
	for key, value := range items {
		err = s.encoder.Encode(&Metrics{
			ID:          string(key),
			ActualValue: value.GetValue(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *HybridStorage) Restore(ctx context.Context) error {
	if !s.backupActive {
		return nil
	}
	s.backupActive = false
	for s.decoder.More() {
		var metric Metrics
		err := s.decoder.Decode(&metric)
		if err != nil {
			return err
		}
		switch metric.ActualValue.(type) {
		case int64:
			err := s.Insert(ctx, MetricName(metric.ID), NewMetricCounter(metric.ActualValue))
			if err != nil {
				return err
			}
		case float64:
			err := s.Insert(ctx, MetricName(metric.ID), NewMetricGauge(metric.ActualValue))
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("metric type is unknown")
		}
	}
	s.backupActive = true
	return nil
}

func (s *HybridStorage) Close(ctx context.Context) error {
	if s.backupActive {
		fmt.Println("make final backup before shutdown")
		if err := s.Backup(ctx); err != nil {
			return err
		}
	}
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func (s *HybridStorage) BatchUpdate(ctx context.Context, metrics []Metrics) error {
	err := s.MemStorage.BatchUpdate(ctx, metrics)
	if err != nil {
		return err
	}
	return s.backupCaller(ctx)
}
