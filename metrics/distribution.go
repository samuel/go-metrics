// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"fmt"
	"math"
	"strconv"
	"sync"
)

type DistributionValue struct {
	Count    uint64
	Sum      float64
	Min      float64
	Max      float64
	Variance float64
}

func (v DistributionValue) Mean() float64 {
	if v.Count > 0 {
		return v.Sum / float64(v.Count)
	}
	return 0.0
}

type DistributionMetric interface {
	Value() DistributionValue
}

type variance struct {
	m float64
	s float64
}

// Distribution tracks the min, max, sum, count, and variance/stddev of a set of values.
type Distribution struct {
	count    uint64
	sum      float64
	min      float64
	max      float64
	variance variance
	mu       sync.Mutex
}

// NewDistribution returns a new instance of an Distribution
func NewDistribution() *Distribution {
	d := &Distribution{}
	d.Reset()
	return d
}

func (d *Distribution) String() string {
	v := d.Value()
	return fmt.Sprintf("{\"count\":%d,\"sum\":%s,\"min\":%s,\"max\":%s,\"stddev\":%s}",
		v.Count,
		strconv.FormatFloat(v.Sum, 'g', -1, 64),
		strconv.FormatFloat(v.Min, 'g', -1, 64),
		strconv.FormatFloat(v.Max, 'g', -1, 64),
		strconv.FormatFloat(math.Sqrt(v.Variance), 'g', -1, 64))
}

func (d *Distribution) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Distribution) MarshalText() ([]byte, error) {
	return d.MarshalJSON()
}

// Reset the distribution to its initial empty state.
func (d *Distribution) Reset() {
	d.mu.Lock()
	d.count = 0
	d.sum = 0
	d.min = math.Inf(1)
	d.max = math.Inf(-1)
	d.variance = variance{m: -1, s: 0}
	d.mu.Unlock()
}

// Update inserts a new data point
func (d *Distribution) Update(value float64) {
	d.mu.Lock()
	d.count++
	d.sum += value
	if value < d.min {
		d.min = value
	}
	if value > d.max {
		d.max = value
	}
	if d.variance.m == -1 {
		d.variance = variance{m: value, s: 0}
	} else {
		newM := d.variance.m + ((value - d.variance.m) / float64(d.count))
		d.variance = variance{
			m: newM,
			s: d.variance.s + ((value - d.variance.m) * (value - newM)),
		}
	}
	d.mu.Unlock()
}

// Count returns the number of data points
func (d *Distribution) Count() uint64 {
	d.mu.Lock()
	v := d.count
	d.mu.Unlock()
	return v
}

// Sum returns the sum of all data points
func (d *Distribution) Sum() float64 {
	d.mu.Lock()
	v := d.sum
	d.mu.Unlock()
	return v
}

// Min returns the minimum value of all data points
func (d *Distribution) Min() float64 {
	d.mu.Lock()
	v := d.min
	if d.count == 0 {
		v = 0.0
	}
	d.mu.Unlock()
	return v
}

// Max returns the maximum value of all data points
func (d *Distribution) Max() float64 {
	d.mu.Lock()
	v := d.max
	if d.count == 0 {
		v = 0.0
	}
	d.mu.Unlock()
	return v
}

// Mean returns the average of all of all data points
func (d *Distribution) Mean() float64 {
	d.mu.Lock()
	v := 0.0
	if d.count != 0 {
		v = float64(d.sum) / float64(d.count)
	}
	d.mu.Unlock()
	return v
}

// Variance returns the variance of all data points
func (d *Distribution) Variance() float64 {
	d.mu.Lock()
	v := 0.0
	if d.count > 1 {
		v = d.variance.s / float64(d.count-1)
	}
	d.mu.Unlock()
	return v
}

// StdDev returns the standard deviation of all data points
func (d *Distribution) StdDev() float64 {
	return math.Sqrt(d.Variance())
}

func (d *Distribution) Value() DistributionValue {
	d.mu.Lock()
	v := DistributionValue{
		Count: d.count,
		Sum:   d.sum,
		Min:   d.min,
		Max:   d.max,
	}
	if d.count > 1 {
		v.Variance = d.variance.s / float64(d.count-1)
	} else if d.count == 0 {
		v.Min = 0.0
		v.Max = 0.0
	}
	d.mu.Unlock()
	return v
}
