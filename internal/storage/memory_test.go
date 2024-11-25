package storage

import (
	"context"
	"reflect"
	"testing"
)

func TestMemStorage_Get(t *testing.T) {
	s := MemStorage{
		Values: map[MetricName]Metric{
			"test1": NewMetricCounter(int64(1)),
			"test2": NewMetricGauge(float64(0.01)),
		},
	}
	ctx := context.Background()

	type args struct {
		key MetricName
	}
	tests := []struct {
		name    string
		args    args
		want    Metric
		wantErr bool
	}{
		{
			name: "get counter metric",
			args: args{
				key: "test1",
			},
			want: NewMetricCounter(int64(1)),
		},
		{
			name: "get gauge metric",
			args: args{
				key: "test2",
			},
			want: NewMetricGauge(float64(0.01)),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.Get(ctx, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("MemStorage.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MemStorage.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemStorage_Insert(t *testing.T) {
	s := NewMemStorage()
	ctx := context.Background()

	type args struct {
		key MetricName
		m   Metric
	}
	tests := []struct {
		name    string
		args    args
		want    Metric
		wantErr bool
	}{
		{
			name: "insert counter metric",
			args: args{
				key: "test1",
				m:   NewMetricCounter(int64(1)),
			},
			want: NewMetricCounter(int64(1)),
		},
		{
			name: "insert gauge metric",
			args: args{
				key: "test2",
				m:   NewMetricGauge(float64(0.01)),
			},
			want: NewMetricGauge(float64(0.01)),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := s.Insert(ctx, tt.args.key, tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("MemStorage.Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemStorage_Update(t *testing.T) {
	s := MemStorage{
		Values: map[MetricName]Metric{
			"test1": NewMetricCounter(int64(1)),
			"test2": NewMetricGauge(float64(0.01)),
		},
	}
	ctx := context.Background()

	type args struct {
		key MetricName
		m   Metric
		v   interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    Metric
		wantErr bool
	}{
		{
			name: "get counter metric",
			args: args{
				key: "test1",
				m:   NewMetricCounter(int64(1)),
				v:   int64(1),
			},
			want: NewMetricCounter(int64(1)),
		},
		{
			name: "get gauge metric",
			args: args{
				key: "test2",
				m:   NewMetricGauge(float64(0.01)),
				v:   float64(0.01),
			},
			want: NewMetricGauge(float64(0.01)),
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := s.Update(ctx, tt.args.key, tt.args.v, tt.args.m); (err != nil) != tt.wantErr {
				t.Errorf("MemStorage.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemStorage_GetAll(t *testing.T) {
	s := MemStorage{
		Values: map[MetricName]Metric{
			"test1": NewMetricCounter(int64(1)),
			"test2": NewMetricGauge(float64(0.01)),
		},
	}
	want := map[MetricName]Metric{
		"test1": NewMetricCounter(int64(1)),
		"test2": NewMetricGauge(float64(0.01)),
	}
	ctx := context.Background()
	got, err := s.GetAll(ctx)
	if err != nil {
		t.Errorf("MemStorage.GetAll() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MemStorage.GetAll() = %v, want %v", got, want)
	}

}

func TestMemStorage_BatchUpdate(t *testing.T) {

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

	s := NewMemStorage()

	if err := s.BatchUpdate(context.Background(), metrics); err != nil {
		t.Errorf("MemStorage.BatchUpdate() error = %v", err)
	}

}
