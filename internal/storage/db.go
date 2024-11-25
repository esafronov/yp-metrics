package storage

import (
	"context"
	"database/sql"
	"fmt"
)

const tableName string = "metrics"

type DBStorage struct {
	db *sql.DB
}

func NewDBStorage(ctx context.Context, db *sql.DB) (*DBStorage, error) {
	storage := &DBStorage{
		db,
	}
	if err := storage.createTable(ctx); err != nil {
		return nil, err
	}
	return storage, nil
}

func (s *DBStorage) Get(ctx context.Context, key MetricName) (Metric, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT value_gauge, value_counter FROM "+tableName+
		" WHERE metric_name = $1 LIMIT 1", string(key))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("metric value is null")
}

func (s *DBStorage) Insert(ctx context.Context, key MetricName, m Metric) error {
	switch m.(type) {
	case *MetricCounter:
		val := m.GetValue().(int64)
		_, err := s.db.ExecContext(ctx, "INSERT INTO "+tableName+"(metric_name, metric_type, value_counter) VALUES ($1,$2,$3) ON CONFLICT (metric_name) DO UPDATE SET value_counter=EXCLUDED.value_counter+$3", string(key), MetricTypeCounter, val)
		if err != nil {
			return err
		}
	case *MetricGauge:
		val := m.GetValue().(float64)
		_, err := s.db.ExecContext(ctx, "INSERT INTO "+tableName+"(metric_name, metric_type, value_gauge) VALUES ($1,$2,$3) ON CONFLICT (metric_name) DO UPDATE SET value_gauge=$3", string(key), MetricTypeGauge, val)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *DBStorage) Update(ctx context.Context, key MetricName, v interface{}, metric Metric) error {
	switch val := v.(type) {
	case int64:
		_, err := s.db.ExecContext(ctx, "UPDATE "+tableName+" SET value_counter=value_counter+$1 WHERE metric_name=$2", val, string(key))
		if err != nil {
			return err
		}
	case float64:
		_, err := s.db.ExecContext(ctx, "UPDATE "+tableName+" SET value_gauge=$1 WHERE metric_name=$2", val, string(key))
		if err != nil {
			return err
		}
	}
	metric.UpdateValue(v)
	return nil
}

func (s *DBStorage) BatchUpdate(ctx context.Context, metrics []Metrics) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmCount, err := tx.PrepareContext(ctx, "SELECT COUNT(*) FROM "+tableName+" WHERE metric_name=$1")
	if err != nil {
		return err
	}
	stmUpdGauge, err := tx.PrepareContext(ctx, "UPDATE "+tableName+" SET value_gauge=$1 WHERE metric_name=$2")
	if err != nil {
		return err
	}
	stmUpdCounter, err := tx.PrepareContext(ctx, "UPDATE "+tableName+" SET value_counter=value_counter+$1 WHERE metric_name=$2")
	if err != nil {
		return err
	}
	stmInsGauge, err := tx.PrepareContext(ctx, "INSERT INTO "+tableName+"(metric_name, metric_type, value_gauge) VALUES ($1, $2, $3) ON CONFLICT (metric_name) DO UPDATE SET value_gauge=$3")
	if err != nil {
		return err
	}
	stmInsCounter, err := tx.PrepareContext(ctx, "INSERT INTO "+tableName+"(metric_name, metric_type, value_counter) VALUES ($1, $2, $3) ON CONFLICT (metric_name) DO UPDATE SET value_counter=EXCLUDED.value_counter+$3")
	if err != nil {
		return err
	}
	var count int
	for _, m := range metrics {
		value := m.ActualValue
		row := stmCount.QueryRowContext(ctx, m.ID)
		if err := row.Scan(&count); err != nil {
			return err
		}
		switch val := value.(type) {
		case int64:
			if count > 0 {
				_, err = stmUpdCounter.ExecContext(ctx, val, m.ID)
			} else {
				_, err = stmInsCounter.ExecContext(ctx, m.ID, m.MType, val)
			}
		case float64:
			if count > 0 {
				_, err = stmUpdGauge.ExecContext(ctx, val, m.ID)
			} else {
				_, err = stmInsGauge.ExecContext(ctx, m.ID, m.MType, val)
			}
		default:
			err = fmt.Errorf("metric type unknown in batch update")
		}
		if err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *DBStorage) GetAll(ctx context.Context) (map[MetricName]Metric, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT metric_name, value_gauge, value_counter FROM "+tableName)
	if err != nil {
		return nil, err
	}
	// обязательно закрываем перед возвратом функции
	defer rows.Close()
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
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func (s *DBStorage) createTable(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// roll back if commit will fail
	defer tx.Rollback()
	tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS `+
		tableName+
		`(
			id SERIAL,
			metric_name VARCHAR(30) NOT NULL,
			metric_type VARCHAR(7) NOT NULL,
			value_gauge DOUBLE PRECISION DEFAULT NULL,
			value_counter BIGINT DEFAULT NULL
		)`)
	tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS metric_name ON `+tableName+` (metric_name)`)
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *DBStorage) Close(ctx context.Context) error {
	err := s.db.Close()
	if err != nil {
		return err
	}
	return nil
}
