package retry

import (
	"net/http"
	"time"
)

// http request with retries
func DoRequest(req *http.Request) (res *http.Response, err error) {
	for n, t := range retriesSchedule {
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			if n < len(retriesSchedule)-1 {
				//fmt.Printf("can't connect to server, retry after %s\r\n", t)
				time.Sleep(t)
			}
			continue
		}
		break
	}
	return
}
