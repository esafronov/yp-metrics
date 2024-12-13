// Agent app executable
package main

import (
	"fmt"

	"github.com/esafronov/yp-metrics/internal/agent"
)

var buildVersion string = "N/A"
var buildDate string = "N/A"
var buildCommit string = "N/A"

func main() {
	fmt.Println(buildVersion)
	fmt.Println(buildDate)
	fmt.Println(buildCommit)
	agent.Run()
}
