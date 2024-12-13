// Package staticcheck contains all analysers from package staticcheck with code prefix SA plus S1003
package staticcheck

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/staticcheck"
)

var Analyzers []*analysis.Analyzer

func init() {
	for _, v := range staticcheck.Analyzers {
		// append all analysers with code prefix SA
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			Analyzers = append(Analyzers, v.Analyzer)
		}
		//append S1003 analyser (Replace call to strings.Index with strings.Contains)
		if v.Analyzer.Name == "S1003" {
			Analyzers = append(Analyzers, v.Analyzer)
		}
	}
}
