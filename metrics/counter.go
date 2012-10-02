package metrics

import (
	"strconv"
	"sync/atomic"
)

type Counter int64

func NewCounter() Counter {
	return 0
}

func (c *Counter) Inc(delta int64) {
	atomic.AddInt64((*int64)(c), delta)
}

func (c *Counter) Dec(delta int64) {
	atomic.AddInt64((*int64)(c), -delta)
}

func (c *Counter) Set(value int64) {
	atomic.StoreInt64((*int64)(c), value)
}

func (c *Counter) Count() int64 {
	return atomic.LoadInt64((*int64)(c))
}

func (c Counter) String() string {
	return strconv.FormatInt(c.Count(), 10)
}
