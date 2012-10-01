package metrics

import (
	"strconv"
	"sync/atomic"
)

type Counter struct {
	count int64
}

func NewCounter() *Counter {
	return &Counter{}
}

func (c *Counter) Inc(delta int64) {
	atomic.AddInt64(&c.count, delta)
}

func (c *Counter) Dec(delta int64) {
	atomic.AddInt64(&c.count, -delta)
}

func (c *Counter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

func (c *Counter) String() string {
	return strconv.FormatInt(c.Count(), 10)
}
