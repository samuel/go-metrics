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

// NewUniformSample returns a sample randomly selects from a stream. Uses Vitter's
// Algorithm R to produce a statistically representative sample.
//
// http://www.cs.umd.edu/~samir/498/vitter.pdf - Random Sampling with a Reservoir
func NewUniformSample(reservoirSize int) Sample {
	return &uniformSample{reservoirSize, make([]int64, reservoirSize), 0}
}

func (sample *uniformSample) Clear() {
	sample.count = 0
}

func (sample *uniformSample) Len() int {
	if sample.count < sample.reservoirSize {
		return sample.count
	}
	return sample.reservoirSize
}

func (sample *uniformSample) Update(value int64) {
	sample.count++
	if sample.count <= sample.reservoirSize {
		sample.values[sample.count-1] = value
	} else {
		r := int(rand.Float64() * float64(sample.count))
		if r < sample.reservoirSize {
			sample.values[r] = value
		}
	}
}

func (sample *uniformSample) Values() []int64 {
	return sample.values[:minInt(sample.count, sample.reservoirSize)]
}
