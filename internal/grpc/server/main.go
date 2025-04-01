// Package server GRPC server based on generated protobuf classes
package server

import (
	// импортируем пакет со сгенерированными protobuf-файлами
	"context"
	"errors"

	pb "github.com/esafronov/yp-metrics/internal/grpc/proto"
	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/pg"
	"github.com/esafronov/yp-metrics/internal/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MetricsServer struct {
	// нужно встраивать тип pb.Unimplemented<TypeName>
	// для совместимости с будущими версиями
	pb.UnimplementedMetricsServer
	Storage       storage.Repositories //repository
	secretKey     string               //secret key for request signature validation
	cryptoKey     string               //RSA private key for decrypting request
	trustedSubnet string               //trusted subnet
}

// NewMetricsServer is factory method
func NewMetricsServer(s storage.Repositories, opts ...func(s *MetricsServer)) *MetricsServer {
	h := &MetricsServer{Storage: s}
	for _, f := range opts {
		f(h)
	}
	return h
}

// OptionWithSecretKey option function to configure MetricsServer to use secretKey
func OptionWithSecretKey(secretKey string) func(s *MetricsServer) {
	return func(s *MetricsServer) {
		s.secretKey = secretKey
	}
}

// OptionWithCryptoKey option function to configure MetricsServer to use cryptoKey
func OptionWithCryptoKey(cryptoKey string) func(s *MetricsServer) {
	return func(s *MetricsServer) {
		s.cryptoKey = cryptoKey
	}
}

// OptionWithTrustedSubnet option function to configure MetricsServer to use trusted subnet
func OptionWithTrustedSubnet(trustedSubnet string) func(s *MetricsServer) {
	return func(s *MetricsServer) {
		s.trustedSubnet = trustedSubnet
	}
}

func (s *MetricsServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	res := &pb.PingResponse{}
	if err := pg.DB.PingContext(ctx); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return res, nil
}

func (s *MetricsServer) List(req *pb.ListRequest, stream pb.Metrics_ListServer) error {
	metrics, err := s.Storage.GetAll(stream.Context())
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	for metricName, m := range metrics {
		var pbMetric = &pb.Metric{
			Id: string(metricName),
		}
		switch tm := m.(type) {
		case *storage.MetricCounter:
			pbMetric.Type = pb.Metric_COUNTER
			val, _ := tm.GetValue().(int64)
			pbMetric.Delta = val
		case *storage.MetricGauge:
			pbMetric.Type = pb.Metric_GAUGE
			val, _ := tm.GetValue().(float64)
			pbMetric.Value = val
		default:
			return status.Errorf(codes.Internal, "type of metric is unknown")
		}
		err := stream.Send(pbMetric)
		if err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
	return nil
}

func (s *MetricsServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	res := &pb.GetResponse{}
	if req.Name == "" {
		return nil, status.Errorf(codes.NotFound, "metric is not found")
	}
	metricName := storage.MetricName(req.Name)
	m, err := s.Storage.Get(ctx, metricName)
	if err != nil {
		logger.Log.Error("get metric", zap.Error(err))
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if m == nil {
		return nil, status.Errorf(codes.NotFound, "metric is not found")
	}
	var pbMetric = &pb.Metric{
		Id: string(metricName),
	}
	switch tm := m.(type) {
	case *storage.MetricCounter:
		pbMetric.Type = pb.Metric_COUNTER
		val, _ := tm.GetValue().(int64)
		pbMetric.Delta = val
	case *storage.MetricGauge:
		pbMetric.Type = pb.Metric_GAUGE
		val, _ := tm.GetValue().(float64)
		pbMetric.Value = val
	default:
		return nil, status.Errorf(codes.Internal, "type of metric is unknown")
	}
	res.Metric = pbMetric
	return res, nil
}

func (s *MetricsServer) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	metricName := storage.MetricName(req.Metric.Id)
	if metricName == "" {
		return nil, status.Errorf(codes.NotFound, "metric is not found")
	}
	m, err := s.Storage.Get(ctx, metricName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	var value any
	switch req.Metric.Type {
	case pb.Metric_COUNTER:
		/*		if req.Metric.Delta == nil {
					return nil, status.Errorf(codes.Internal, "metric delta is nil")
				}
		*/
		value = req.Metric.GetDelta()
	case pb.Metric_GAUGE:
		/*		if req.Metric.Delta == nil {
					return nil, status.Errorf(codes.Internal, "metric value is nil")
				}
		*/
		value = req.Metric.GetValue()
	default:
		return nil, status.Errorf(codes.InvalidArgument, "metric type is wrong %s", req.Metric.Type)
	}
	if m != nil {
		err = s.Storage.Update(ctx, metricName, value, m)
		if err != nil {
			logger.Log.Error("update metric", zap.Error(err))
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	} else {
		switch req.Metric.Type {
		case pb.Metric_COUNTER:
			m = storage.NewMetricCounter(value)
		case pb.Metric_GAUGE:
			m = storage.NewMetricGauge(value)
		default:
			err = errors.New("unknown metric type")
			logger.Log.Error(err.Error(), zap.Error(err))
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		err = s.Storage.Insert(ctx, metricName, m)
		if err != nil {
			logger.Log.Error("insert metric", zap.Error(err))
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}
	res := &pb.UpdateResponse{
		Metric: req.Metric,
	}
	if req.Metric.Type == pb.Metric_COUNTER {
		val, _ := m.GetValue().(int64)
		res.Metric.Delta = val
	}
	return res, nil
}

func (s *MetricsServer) BatchUpdate(ctx context.Context, req *pb.BatchUpdateRequest) (*pb.BatchUpdateResponse, error) {
	var metrics []storage.Metrics
	for _, m := range req.GetMetric() {
		var metricType storage.MetricType
		var val any
		switch m.Type {
		case pb.Metric_COUNTER:
			metricType = storage.MetricTypeCounter
			val = m.Delta
		case pb.Metric_GAUGE:
			metricType = storage.MetricTypeGauge
			val = m.Value
		default:
			err := errors.New("unknown metric type")
			logger.Log.Error(err.Error(), zap.Error(err))
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		m := storage.Metrics{
			ID:          m.Id,
			MType:       string(metricType),
			ActualValue: val,
		}
		metrics = append(metrics, m)
	}
	err := s.Storage.BatchUpdate(context.Background(), metrics)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return &pb.BatchUpdateResponse{}, nil
}
