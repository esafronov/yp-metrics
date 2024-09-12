package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const tableName string = "metrics"

type DbStorage struct {
	db *sql.DB
}

func NewDbStorage(ctx context.Context, db *sql.DB) (*DbStorage, error) {
	storage := &DbStorage{
		db,
	}
	if err := storage.createTable(ctx); err != nil {
		return nil, err
	}
	return storage, nil
}

func (s *DbStorage) Get(ctx context.Context, key MetricName) (Metric, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT value_gauge, value_counter FROM "+tableName+
		" WHERE metric_name = $1 LIMIT 1", string(key))
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		return nil, nil
	}
	var gaugeValue sql.NullFloat64
	var counterValue sql.NullInt64
	err = rows.Scan(&gaugeValue, &counterValue)
	if err != nil {
		return nil, err
	}
	//test gauge value is not null
	if gaugeValue.Valid {
		v := gaugeValue.Float64
		return NewMetricGauge(v), nil
	}
	//test counter value is not null
	if counterValue.Valid {
		v := counterValue.Int64
		return NewMetricCounter(v), nil
	}
	return nil, fmt.Errorf("metric value is null")
}

func (s *DbStorage) Insert(ctx context.Context, key MetricName, m Metric) error {
	switch m.(type) {
	case *MetricCounter:
		val := m.GetValue().(int64)
		_, err := s.db.ExecContext(ctx, "INSERT INTO "+tableName+"(metric_name, metric_type, value_counter) VALUES ($1, $2, $3)", string(key), MetricTypeCounter, val)
		if err != nil {
			return err
		}
	case *MetricGauge:
		val := m.GetValue().(float64)
		_, err := s.db.ExecContext(ctx, "INSERT INTO "+tableName+"(metric_name, metric_type, value_gauge) VALUES ($1,$2,$3)", string(key), MetricTypeGauge, val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DbStorage) Update(ctx context.Context, key MetricName, v interface{}, metric Metric) error {
	switch v.(type) {
	case int64:
		val := v.(int64)
		_, err := s.db.ExecContext(ctx, "UPDATE "+tableName+" SET value_counter=value_counter+$1 WHERE metric_name=$2", val, string(key))
		if err != nil {
			return err
		}
	case float64:
		val := v.(float64)
		_, err := s.db.ExecContext(ctx, "UPDATE "+tableName+" SET value_gauge=$1 WHERE metric_name=$2", val, string(key))
		if err != nil {
			return err
		}
	}
	metric.UpdateValue(v)
	return nil
}

func (s *DbStorage) GetAll(ctx context.Context) (map[MetricName]Metric, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT metric_name, value_gauge, value_counter FROM "+tableName)
	if err != nil {
		return nil, err
	}
	var gaugeValue sql.NullFloat64
	var counterValue sql.NullInt64
	var metricName string
	metrics := map[MetricName]Metric{}
	for rows.Next() {
		err := rows.Scan(&metricName, &gaugeValue, &counterValue)
		if err != nil {
			return nil, err
		}
		//test gauge value is not null
		if gaugeValue.Valid {
			v := gaugeValue.Float64
			metrics[MetricName(metricName)] = NewMetricGauge(v)
		}
		//test counter value is not null
		if counterValue.Valid {
			v := counterValue.Int64
			metrics[MetricName(metricName)] = NewMetricCounter(v)
		}
	}
	return metrics, nil
}

func (s *DbStorage) createTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS `+
		tableName+
		`(
			id SERIAL,
			metric_name VARCHAR(30) NOT NULL,
			metric_type VARCHAR(7) NOT NULL,
			value_gauge DOUBLE PRECISION DEFAULT NULL,
			value_counter BIGINT DEFAULT NULL
		)`)
	if err != nil {
		return fmt.Errorf("create table %v", err)
	}
	return nil
}

func (s *DbStorage) Close(ctx context.Context) error {
	err := s.db.Close()
	if err != nil {
		return err
	}
	return nil
}
