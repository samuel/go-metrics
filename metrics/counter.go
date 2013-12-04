// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"strconv"
	"sync/atomic"
)

type CounterValue int64

type Counter interface {
	Inc(delta int64)
	Count() int64
	String() string
}

type atomicCounter int64

func NewCounter() Counter {
	c := atomicCounter(int64(0))
	return &c
}

func (c *atomicCounter) Inc(delta int64) {
	atomic.AddInt64((*int64)(c), delta)
}

func (c *atomicCounter) Count() int64 {
	return atomic.LoadInt64((*int64)(c))
}

func (c *atomicCounter) String() string {
	return strconv.FormatInt(c.Count(), 10)
}
