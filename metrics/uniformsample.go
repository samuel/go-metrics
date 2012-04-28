// A random sample of a stream. Uses Vitter's Algorithm R to produce a
// statistically representative sample.
// 
// http://www.cs.umd.edu/~samir/498/vitter.pdf - Random Sampling with a Reservoir

package metrics

import (
	"math/rand"
)

type UniformSample struct {
	reservoirSize int
	values        []float64
	count         int
}

func NewUniformSample(reservoirSize int) *UniformSample {
	return &UniformSample{reservoirSize, make([]float64, reservoirSize), 0}
}

func (self *UniformSample) Clear() {
	self.count = 0
}

func (self *UniformSample) Len() int {
	if self.count < self.reservoirSize {
		return self.count
	}
	return self.reservoirSize
}

func (self *UniformSample) Update(value float64) {
	self.count += 1
	if self.count <= self.reservoirSize {
		self.values[self.count-1] = value
	} else {
		r := int(rand.Float64() * float64(self.count))
		if r < self.reservoirSize {
			self.values[r] = value
		}
	}
}

func (self *UniformSample) Values() []float64 {
	return self.values[:minInt(self.count, self.reservoirSize)]
}
