// Server app executable
package main

import (
	"fmt"
	"log"

	"github.com/esafronov/yp-metrics/internal/server"
)

var buildVersion string = "N/A"
var buildDate string = "N/A"
var buildCommit string = "N/A"

func main() {
	fmt.Println(buildVersion)
	fmt.Println(buildDate)
	fmt.Println(buildCommit)
	log.Fatal(server.Run())
}
