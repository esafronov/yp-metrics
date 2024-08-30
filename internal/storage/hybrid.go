package storage

import (
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

func NewHybridStorage(filename string, storeInterval int, restore *bool) (storage *HybridStorage, err error) {
	var file *os.File
	//if filename is not empty we open it
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
		storeInterval: storeInterval,
		lastStored:    time.Time{},
		encoder:       encoder,
		decoder:       decoder,
		backupActive:  backupActive,
	}
	if *restore {
		err = storage.Restore()
	}
	return
}

func (s *HybridStorage) Insert(key MetricName, m Metric) {
	s.MemStorage.Insert(key, m)
	s.backupCaller()
}

func (s *HybridStorage) Update(key MetricName, v interface{}) {
	s.MemStorage.Update(key, v)
	s.backupCaller()
}

func (s *HybridStorage) backupCaller() {
	if !s.backupActive {
		return
	}
	if time.Since(s.lastStored).Seconds() > float64(s.storeInterval) {
		s.Backup()
	}
}

func (s *HybridStorage) Backup() error {
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
	for key, value := range s.GetAll() {
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

func (s *HybridStorage) Restore() error {
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
			s.Insert(MetricName(metric.ID), NewMetricCounter(metric.ActualValue))
		case float64:
			s.Insert(MetricName(metric.ID), NewMetricGauge(metric.ActualValue))
		default:
			return fmt.Errorf("metric type is unknown")
		}
	}
	s.backupActive = true
	return nil
}

func (s *HybridStorage) Close() error {
	return s.file.Close()
}
