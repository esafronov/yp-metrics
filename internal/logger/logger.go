// Package logger includes singleton and http middleware for logging requests
package logger

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(level string) error {
	// преобразуем текстовый уровень логирования в zap.AtomicLevel
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	cfg := zap.NewDevelopmentConfig()
	// устанавливаем уровень
	cfg.Level = lvl
	// создаём логер на основе конфигурации
	lz, err := cfg.Build()
	if err != nil {
		return err
	}
	defer func(l *zap.Logger) {
		err := l.Sync()
		if err != nil {
			_, isPathErr := err.(*fs.PathError)
			if !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTTY) && !isPathErr {
				panic(err)
			}
		}
	}(lz)
	Log = lz
	return nil
}

// RequestLogger middleware for logging incomming requests
func RequestLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeStamp := time.Now()
		rd := &ResponseData{
			size:   0,
			status: 0,
		}
		wr := &LoggerResponseWriter{
			data: rd,
			rw:   w,
		}
		h.ServeHTTP(wr, r)
		duration := time.Since(timeStamp).Milliseconds()
		Log.Info("request",
			zap.String("method", r.Method),
			zap.String("URI", r.RequestURI),
			zap.Int64("duration", duration),
			zap.Int("status", rd.status),
			zap.Int("size", rd.size))
	})
}

type ResponseData struct {
	size   int
	status int
}

type LoggerResponseWriter struct {
	rw   http.ResponseWriter
	data *ResponseData
}

func (w *LoggerResponseWriter) Write(b []byte) (int, error) {
	len, err := w.rw.Write(b)
	w.data.size += len
	return len, err
}

func (w *LoggerResponseWriter) Header() http.Header {
	return w.rw.Header()
}

func (w *LoggerResponseWriter) WriteHeader(statusCode int) {
	w.rw.WriteHeader(statusCode)
	w.data.status = statusCode
}

// UnaryLoggerInterceptor is the interceptor for gRPC server for logging remote calls
func UnaryLoggerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	timeStamp := time.Now()
	res, err := handler(ctx, req)
	duration := time.Since(timeStamp).Milliseconds()
	Log.Info("request",
		zap.String("method", info.FullMethod),
		zap.Int64("duration", duration),
	)
	return res, err
}
