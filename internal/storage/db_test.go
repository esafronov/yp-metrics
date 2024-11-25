package storage

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDBStorage_Get(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	s := &DBStorage{
		db: db,
	}
	require.NoError(t, err)

	type arg struct {
		key MetricName
		t   MetricType
		v   interface{}
	}

	type want struct {
		m Metric
	}

	tests := []struct {
		name    string
		arg     arg
		want    want
		wantErr bool
	}{
		{
			name: "get counter value",
			arg: arg{
				key: MetricName("test"),
				t:   MetricTypeCounter,
				v:   int64(1),
			},
			want: want{
				m: NewMetricCounter(int64(1)),
			},
		},
		{
			name: "get gauge value",
			arg: arg{
				key: MetricName("gtest"),
				t:   MetricTypeGauge,
				v:   float64(1.1),
			},
			want: want{
				m: NewMetricGauge(float64(1.1)),
			},
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := sqlmock.NewRows([]string{"value_gauge", "value_counter"})

			if tt.arg.t == MetricTypeCounter {
				rows.AddRow(nil, tt.arg.v.(int64))
			}

			if tt.arg.t == MetricTypeGauge {
				rows.AddRow(tt.arg.v.(float64), nil)
			}

			mock.ExpectQuery("^SELECT value_gauge, value_counter").
				WithArgs(tt.arg.key).
				WillReturnRows(rows)

			m, err := s.Get(ctx, tt.arg.key)
			if err != nil {
				t.Errorf("error was not expected : %s", err)
			}
			require.Equal(t, tt.want.m, m, "метрика в хранилище не соответствует ожидаемой")
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}

}

func TestDBStorage_Insert(t *testing.T) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	ctx := context.Background()
	s := &DBStorage{
		db: db,
	}
	require.NoError(t, err)

	type arg struct {
		key MetricName
		v   Metric
		t   string
	}

	type want struct {
		lastID   int64
		effected int64
	}

	tests := []struct {
		name    string
		arg     arg
		want    want
		wantErr bool
	}{
		{
			name: "successfull insert counter",
			arg: arg{
				key: MetricName("test"),
				v:   NewMetricCounter(int64(3)),
				t:   "counter",
			},
			want: want{1, 1},
		},
		{
			name: "successfull insert gauge",
			arg: arg{
				key: MetricName("gtest"),
				v:   NewMetricGauge(float64(3)),
				t:   "gauge",
			},
			want: want{2, 1},
		},
		{
			name: "successfull update counter",
			arg: arg{
				key: MetricName("test"),
				v:   NewMetricCounter(int64(2)),
				t:   "counter",
			},
			want: want{2, 1},
		},
		{
			name: "successfull update gauge",
			arg: arg{
				key: MetricName("gtest"),
				v:   NewMetricGauge(float64(10)),
				t:   "gauge",
			},
			want: want{2, 1},
		},
		// TODO: Add test cases.

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			v := tt.arg.v.GetValue()
			mock.ExpectExec("^INSERT INTO").
				WithArgs(tt.arg.key, tt.arg.t, v).
				WillReturnResult(sqlmock.NewResult(tt.want.lastID, tt.want.effected))

			if err := s.Insert(ctx, tt.arg.key, tt.arg.v); err != nil {
				t.Errorf("error was not expected : %s", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}

}

func TestDBStorage_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	ctx := context.Background()
	s := &DBStorage{
		db: db,
	}
	require.NoError(t, err)

	type arg struct {
		key MetricName
		v   Metric
	}

	type want struct {
		lastID   int64
		effected int64
	}

	tests := []struct {
		name    string
		arg     arg
		want    want
		wantErr bool
	}{
		{
			name: "successfull update counter",
			arg: arg{
				key: MetricName("test"),
				v:   NewMetricCounter(int64(3)),
			},
			want: want{0, 1},
		},
		{
			name: "successfull update gauge",
			arg: arg{
				key: MetricName("gtest"),
				v:   NewMetricGauge(float64(0.1)),
			},
			want: want{0, 1},
		},

		// TODO: Add test cases.

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			v := tt.arg.v.GetValue()

			mock.ExpectExec("^UPDATE").
				WithArgs(v, tt.arg.key).
				WillReturnResult(sqlmock.NewResult(tt.want.lastID, tt.want.effected))

			if err := s.Update(ctx, tt.arg.key, v, tt.arg.v); err != nil {
				t.Errorf("error was not expected : %s", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestDBStorage_BatchUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	ctx := context.Background()
	require.NoError(t, err)
	s := &DBStorage{
		db: db,
	}

	metrics := []Metrics{{
		ID:          "test",
		MType:       "counter",
		ActualValue: int64(1),
	}, {
		ID:          "test",
		MType:       "counter",
		ActualValue: int64(1),
	}, {
		ID:          "gtest",
		MType:       "gauge",
		ActualValue: float64(0.1),
	}, {
		ID:          "gtest",
		MType:       "gauge",
		ActualValue: float64(0.2),
	}}

	mock.ExpectBegin()
	mock.ExpectPrepare("^SELECT COUNT")
	mock.ExpectPrepare("^UPDATE")
	mock.ExpectPrepare("^UPDATE")
	mock.ExpectPrepare("^INSERT")
	mock.ExpectPrepare("^INSERT")
	mock.ExpectQuery("^SELECT COUNT").
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec("^INSERT INTO").
		WithArgs("test", "counter", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("^SELECT COUNT").
		WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectExec("^UPDATE").
		WithArgs(1, "test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("^SELECT COUNT").
		WithArgs("gtest").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec("^INSERT INTO").
		WithArgs("gtest", "gauge", 0.1).
		WillReturnResult(sqlmock.NewResult(2, 1))

	mock.ExpectQuery("^SELECT COUNT").
		WithArgs("gtest").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectExec("^UPDATE").
		WithArgs(0.2, "gtest").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err = s.BatchUpdate(ctx, metrics); err != nil {
		t.Errorf("error was not expected : %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

}

func TestDBStorage_GetAll(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()
	s := &DBStorage{
		db: db,
	}
	require.NoError(t, err)
	mock.ExpectQuery("SELECT metric_name, value_gauge, value_counter").
		WillReturnRows(mock.NewRows([]string{"metric_name", "value_gauge", "value_counter"}).
			AddRow("test1", nil, 1).
			AddRow("test2", nil, 2).
			AddRow("test3", 0.1, nil))

	metrics, err := s.GetAll(context.Background())
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	if _, ok := metrics["test1"]; !ok {
		t.Errorf("metric test1 is not returned")
	}

	if _, ok := metrics["test2"]; !ok {
		t.Errorf("metric test2 is not returned")
	}

	if _, ok := metrics["test3"]; !ok {
		t.Errorf("metric test2 is not returned")
	}

}
