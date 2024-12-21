// Package pprofserv run pprof server on independent address
package pprofserv

import (
	"net/http"
	"net/http/pprof"

	"github.com/esafronov/yp-metrics/internal/logger"
)

type DebugServer struct {
	http.Server
}

func NewDebugServer(addr string) *DebugServer {
	return &DebugServer{
		Server: http.Server{
			Addr: addr,
		},
	}
}

func (s *DebugServer) Start() {
	go func() {
		// будем отдавать профиль только внутренним пользователям
		debugMux := http.NewServeMux()
		// только runtime pprof
		debugMux.HandleFunc("/debug/pprof/", pprof.Index)
		debugMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		debugMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		debugMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		debugMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		s.Handler = debugMux
		err := s.ListenAndServe()
		if err != nil {
			logger.Log.Info(err.Error())
		}
	}()
}

func (s *DebugServer) Close() {
	err := s.Server.Close()
	if err != nil {
		logger.Log.Info(err.Error())
	}
}
