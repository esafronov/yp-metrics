// Staticlint app executable
package main

import (
	"github.com/esafronov/yp-metrics/internal/staticlint/customcheck"
	"github.com/esafronov/yp-metrics/internal/staticlint/publiccheck"
	"github.com/esafronov/yp-metrics/internal/staticlint/standartcheck"
	"github.com/esafronov/yp-metrics/internal/staticlint/staticcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	var analyzers []*analysis.Analyzer
	//append all standart analysers
	analyzers = append(analyzers, standartcheck.Analyzers...)
	//append static analysers
	analyzers = append(analyzers, staticcheck.Analyzers...)
	//append public analysers
	analyzers = append(analyzers, publiccheck.Analyzers...)
	//append custom analyser
	analyzers = append(analyzers, customcheck.OsExitInMainAnalyzer)
	multichecker.Main(analyzers...)
}
