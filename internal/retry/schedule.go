package retry

import "time"

// 4 tries first one and if not succeed after 1, 3, 5 seconds
var retriesSchedule = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
	0 * time.Second,
}
