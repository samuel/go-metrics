// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"strconv"
	"sync/atomic"
)

type GaugeValue float64

type atomicInt64Gauge int64

type IntegerGauge interface {
	Inc(delta int64)
	Dec(delta int64)
	Set(delta int64)
	Value() int64
	String() string
}

func NewIntegerGauge() IntegerGauge {
	c := atomicInt64Gauge(int64(0))
	return &c
}

func (c *atomicInt64Gauge) Inc(delta int64) {
	atomic.AddInt64((*int64)(c), delta)
}

func (c *atomicInt64Gauge) Dec(delta int64) {
	atomic.AddInt64((*int64)(c), -delta)
}

func (c *atomicInt64Gauge) Set(value int64) {
	atomic.StoreInt64((*int64)(c), value)
}

func (c *atomicInt64Gauge) Value() int64 {
	return atomic.LoadInt64((*int64)(c))
}

func (c *atomicInt64Gauge) String() string {
	return strconv.FormatInt(c.Value(), 10)
}

func (c *atomicInt64Gauge) MarshalJSON() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *atomicInt64Gauge) MarshalText() ([]byte, error) {
	return c.MarshalJSON()
}
