package retry

import (
	"os"
	"time"
)

// OpenFile open file with retries
func OpenFile(filename string) (file *os.File, err error) {
	for n, t := range retriesSchedule {
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			if n < len(retriesSchedule)-1 {
				//fmt.Printf("can't open file %s, retry after %s\r\n", filename, t)
				time.Sleep(t)
			}
			continue
		}
		break
	}
	return
}
