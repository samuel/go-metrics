// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math/rand"
)

type uniformSample struct {
	reservoirSize int
	values        []int64
	count         int
}

// A random sample of a stream. Uses Vitter's Algorithm R to produce a
// statistically representative sample.
//
// http://www.cs.umd.edu/~samir/498/vitter.pdf - Random Sampling with a Reservoir
func NewUniformSample(reservoirSize int) Sample {
	return &uniformSample{reservoirSize, make([]int64, reservoirSize), 0}
}

func (self *uniformSample) Clear() {
	self.count = 0
}

func (self *uniformSample) Len() int {
	if self.count < self.reservoirSize {
		return self.count
	}
	return self.reservoirSize
}

func (self *uniformSample) Update(value int64) {
	self.count++
	if self.count <= self.reservoirSize {
		self.values[self.count-1] = value
	} else {
		r := int(rand.Float64() * float64(self.count))
		if r < self.reservoirSize {
			self.values[r] = value
		}
	}
}

func (self *uniformSample) Values() []int64 {
	return self.values[:minInt(self.count, self.reservoirSize)]
}
