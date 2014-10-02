// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"strconv"
	"sync/atomic"
)

// CounterValue is used for stats reporting to identify the value as a counter rather than a gauge.
type CounterValue uint64

// Counter is the interface for a counter metric.
type Counter interface {
	Inc(delta uint64)
	Count() uint64
	String() string
}

type atomicCounter uint64

// NewCounter returns a counter implemented as an atomic int64.
func NewCounter() Counter {
	c := atomicCounter(uint64(0))
	return &c
}

func (c *atomicCounter) Inc(delta uint64) {
	atomic.AddUint64((*uint64)(c), delta)
}

func (c *atomicCounter) Count() uint64 {
	return atomic.LoadUint64((*uint64)(c))
}

func (c *atomicCounter) Reset() uint64 {
	return atomic.SwapUint64((*uint64)(c), 0)
}

func (c *atomicCounter) String() string {
	return strconv.FormatUint(c.Count(), 10)
}

func (c *atomicCounter) MarshalJSON() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *atomicCounter) MarshaText() ([]byte, error) {
	return c.MarshalJSON()
}
