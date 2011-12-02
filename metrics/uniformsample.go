// A random sample of a stream. Uses Vitter's Algorithm R to produce a
// statistically representative sample.
// 
// http://www.cs.umd.edu/~samir/498/vitter.pdf - Random Sampling with a Reservoir

package metrics

import (
    "rand"
)

type UniformSample struct {
    reservoir_size int
    values         []float64
    count          int
}

func NewUniformSample(reservoir_size int) *UniformSample {
    return &UniformSample{reservoir_size, make([]float64, reservoir_size), 0}
}

func (self *UniformSample) Clear() {
    self.count = 0
}

func (self *UniformSample) Len() int {
    if self.count < self.reservoir_size {
        return self.count
    }
    return self.reservoir_size
}

func (self *UniformSample) Update(value float64) {
    self.count += 1
    if self.count <= self.reservoir_size {
        self.values[self.count-1] = value
    } else {
        r := int(rand.Float64() * float64(self.count))
        if r < self.reservoir_size {
            self.values[r] = value
        }
    }
}

func (self *UniformSample) GetValues() []float64 {
    return self.values[:minInt(self.count, self.reservoir_size)]
}
