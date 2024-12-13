// Package publiccheck contains errcheck and nilaway analyzers
package publiccheck

import (
	"github.com/kisielk/errcheck/errcheck"
	"go.uber.org/nilaway"
	"golang.org/x/tools/go/analysis"
)

var Analyzers []*analysis.Analyzer

func init() {
	Analyzers = append(Analyzers, errcheck.Analyzer, nilaway.Analyzer)
}
