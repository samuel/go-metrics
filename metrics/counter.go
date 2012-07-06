package metrics

import (
	"sync/atomic"
)

type Counter struct {
	count uint64
}

func NewCounter() *Counter {
	return &Counter{}
}

func (c *Counter) Inc(delta uint64) {
	atomic.AddUint64(&c.count, delta)
}

func (c *Counter) Dec(delta uint64) {
	atomic.AddUint64(&c.count, -delta)
}

func (c *Counter) Count() uint64 {
	return atomic.LoadUint64(&c.count)
}
