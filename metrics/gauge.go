// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"strconv"
	"sync/atomic"
)

type GaugeMetric interface {
	Value() float64
}

type GaugeValue float64

func (v GaugeValue) Value() float64 {
	return float64(v)
}

type GaugeFunc func() float64

func (f GaugeFunc) Value() float64 {
	return f()
}

type IntegerGauge struct {
	value int64
}

func NewIntegerGauge() *IntegerGauge {
	return &IntegerGauge{}
}

func (c *IntegerGauge) Inc(delta int64) {
	atomic.AddInt64(&c.value, delta)
}

func (c *IntegerGauge) Dec(delta int64) {
	atomic.AddInt64(&c.value, -delta)
}

func (c *IntegerGauge) Set(value int64) {
	atomic.StoreInt64(&c.value, value)
}

func (c *IntegerGauge) Reset() int64 {
	return atomic.SwapInt64(&c.value, 0)
}

func (c *IntegerGauge) IntegerValue() int64 {
	return atomic.LoadInt64(&c.value)
}

func (c *IntegerGauge) Value() float64 {
	return float64(c.IntegerValue())
}

func (c *IntegerGauge) String() string {
	return strconv.FormatInt(c.IntegerValue(), 10)
}

func (c *IntegerGauge) MarshalJSON() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *IntegerGauge) MarshalText() ([]byte, error) {
	return c.MarshalJSON()
}
