// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"strconv"
	"sync/atomic"
)

// Counter is the interface for a counter metric.
type CounterMetric interface {
	Count() uint64
}

type CounterValue uint64

func (v CounterValue) Count() uint64 {
	return uint64(v)
}

type CounterFunc func() uint64

func (f CounterFunc) Count() uint64 {
	return f()
}

type Counter struct {
	value uint64
}

// NewCounter returns a counter implemented as an atomic uint64.
func NewCounter() *Counter {
	return &Counter{}
}

func (c *Counter) Inc(delta uint64) {
	atomic.AddUint64(&c.value, delta)
}

func (c *Counter) Count() uint64 {
	return atomic.LoadUint64(&c.value)
}

func (c *Counter) Reset() uint64 {
	return atomic.SwapUint64(&c.value, 0)
}

func (c *Counter) String() string {
	return strconv.FormatUint(c.Count(), 10)
}

func (c *Counter) MarshalJSON() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *Counter) MarshaText() ([]byte, error) {
	return c.MarshalJSON()
}
