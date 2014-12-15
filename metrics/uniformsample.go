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
}

// NewUniformSample returns a sample randomly selects from a stream. Uses Vitter's
// Algorithm R to produce a statistically representative sample.
//
// http://www.cs.umd.edu/~samir/498/vitter.pdf - Random Sampling with a Reservoir
func NewUniformSample(reservoirSize int) Sample {
	return &uniformSample{
		reservoirSize: reservoirSize,
		values:        make([]int64, 0, reservoirSize),
	}
}

func (s *uniformSample) Clear() {
	s.values = s.values[:0]
}

func (s *uniformSample) Len() int {
	return len(s.values)
}

func (s *uniformSample) Update(value int64) {
	if len(s.values) < s.reservoirSize {
		s.values = append(s.values, value)
	} else {
		r := int(rand.Float64() * float64(len(s.values)))
		if r < s.reservoirSize {
			s.values[r] = value
		}
	}
}

func (s *uniformSample) Values() []int64 {
	return s.values
}
