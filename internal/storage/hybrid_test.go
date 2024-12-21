package storage

import (
	"context"
	"reflect"
	"testing"
)

func TestHybridStorage_Get(t *testing.T) {
	s := HybridStorage{
		MemStorage: MemStorage{
			Values: map[MetricName]Metric{
				"test1": NewMetricCounter(int64(1)),
				"test2": NewMetricGauge(float64(0.01)),
			},
		},
	}
	ctx := context.Background()

	type args struct {
		key MetricName
	}
	tests := []struct {
		want    Metric
		name    string
		args    args
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
				t.Errorf("HybridStorage.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HybridStorage.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHybridStorage_Insert(t *testing.T) {
	ctx := context.Background()
	filename := ""
	restore := false
	storeInterval := 0

	s, err := NewHybridStorage(ctx, &filename, &storeInterval, &restore)
	if err != nil {
		t.Errorf("NewHybridStorage error = %v", err)
	}

	type args struct {
		m   Metric
		key MetricName
	}
	tests := []struct {
		args    args
		want    Metric
		name    string
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
				t.Errorf("HybridStorage.Insert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHybridStorage_Update(t *testing.T) {
	s := HybridStorage{
		MemStorage: MemStorage{
			Values: map[MetricName]Metric{
				"test1": NewMetricCounter(int64(1)),
				"test2": NewMetricGauge(float64(0.01)),
			},
		},
	}
	ctx := context.Background()

	type args struct {
		m   Metric
		v   interface{}
		key MetricName
	}
	tests := []struct {
		args    args
		want    Metric
		name    string
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
				t.Errorf("HybridStorage.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHybridStorage_GetAll(t *testing.T) {
	s := HybridStorage{
		MemStorage: MemStorage{
			Values: map[MetricName]Metric{
				"test1": NewMetricCounter(int64(1)),
				"test2": NewMetricGauge(float64(0.01)),
			},
		},
	}
	want := map[MetricName]Metric{
		"test1": NewMetricCounter(int64(1)),
		"test2": NewMetricGauge(float64(0.01)),
	}
	ctx := context.Background()
	got, err := s.GetAll(ctx)
	if err != nil {
		t.Errorf("HybridStorage.GetAll() error = %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("HybridStorage.GetAll() = %v, want %v", got, want)
	}

}

func TestHybridStorage_BatchUpdate(t *testing.T) {

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

	ctx := context.Background()
	filename := ""
	restore := false
	storeInterval := 0

	s, err := NewHybridStorage(ctx, &filename, &storeInterval, &restore)
	if err != nil {
		t.Errorf("NewHybridStorage error = %v", err)
	}
	if s == nil {
		t.Errorf("NewHybridStorage is nil")
		return
	}
	if err := s.BatchUpdate(context.Background(), metrics); err != nil {
		t.Errorf("HybridStorage.BatchUpdate() error = %v", err)
	}
}

func TestHybridStorage_Close(t *testing.T) {
	ctx := context.Background()
	filename := ""
	restore := false
	storeInterval := 0
	s, err := NewHybridStorage(ctx, &filename, &storeInterval, &restore)
	if err != nil {
		t.Errorf("NewHybridStorage error = %v", err)
	}
	if s == nil {
		t.Errorf("NewHybridStorage is nil")
		return
	}
	if err := s.Close(context.Background()); err != nil {
		t.Errorf("HybridStorage.BatchUpdate() error = %v", err)
	}
}
