// Server app executable
package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/esafronov/yp-metrics/internal/logger"
	"github.com/esafronov/yp-metrics/internal/server"
	"go.uber.org/zap"
)

var buildVersion string = "N/A"
var buildDate string = "N/A"
var buildCommit string = "N/A"

func main() {
	fmt.Println(buildVersion)
	fmt.Println(buildDate)
	fmt.Println(buildCommit)
	err := logger.Initialize("debug")
	if err != nil {
		panic("can't initialize logger")
	}
	if err := server.Run(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			logger.Log.Info("server shutdown", zap.Error(err))
		} else {
			logger.Log.Error("app error", zap.Error(err))
		}
	}
}
