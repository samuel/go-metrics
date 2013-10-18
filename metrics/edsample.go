// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"container/heap"
	"math"
	"math/rand"
	"time"
)

const (
	edRescaleThreshold = time.Hour
)

// Reservoir

type priorityValue struct {
	priority float64
	value    int64
}

type reservoir struct {
	samples []priorityValue
}

func (r *reservoir) Clear() {
	r.samples = r.samples[:0]
}

func (r *reservoir) Get(i int) priorityValue {
	return r.samples[i]
}

func (r *reservoir) Values() (values []int64) {
	values = make([]int64, len(r.samples))
	for i, sample := range r.samples {
		values[i] = sample.value
	}
	return
}

func (r *reservoir) ScalePriority(scale float64) {
	for i, sample := range r.samples {
		r.samples[i] = priorityValue{sample.priority * scale, sample.value}
	}
}

func (r *reservoir) Len() int {
	return len(r.samples)
}

func (r *reservoir) Less(i, j int) bool {
	return r.samples[i].priority < r.samples[j].priority
}

func (r *reservoir) Swap(i, j int) {
	r.samples[i], r.samples[j] = r.samples[j], r.samples[i]
}

func (r *reservoir) Push(x interface{}) {
	r.samples = append(r.samples, x.(priorityValue))
}

func (r *reservoir) Pop() interface{} {
	v := r.samples[len(r.samples)-1]
	r.samples = r.samples[:len(r.samples)-1]
	return v
}

type exponentiallyDecayingSample struct {
	// the number of samples to keep in the sampling reservoir
	reservoirSize int
	// the exponential decay factor; the higher this is, the more
	// biased the sample will be towards newer values
	alpha         float64
	values        *reservoir
	count         int
	startTime     time.Time
	nextScaleTime time.Time
	now           func() time.Time
}

// An exponentially-decaying random sample of values. Uses Cormode et
// al's forward-decaying priority reservoir sampling method to produce a
// statistically representative sample, exponentially biased towards newer
// entries.
//
// http://www.research.att.com/people/Cormode_Graham/library/publications/CormodeShkapenyukSrivastavaXu09.pdf
// Cormode et al. Forward Decay: A Practical Time Decay Model for Streaming
// Systems. ICDE '09: Proceedings of the 2009 IEEE International Conference on
// Data Engineering (2009)
func NewExponentiallyDecayingSample(reservoirSize int, alpha float64) Sample {
	return NewExponentiallyDecayingSampleWithCustomTime(reservoirSize, alpha, time.Now)
}

func NewExponentiallyDecayingSampleWithCustomTime(reservoirSize int, alpha float64, now func() time.Time) Sample {
	eds := exponentiallyDecayingSample{
		reservoirSize: reservoirSize,
		alpha:         alpha,
		values:        &reservoir{},
		now:           now,
	}
	eds.Clear()
	return &eds
}

func (self *exponentiallyDecayingSample) Clear() {
	self.values.Clear()
	heap.Init(self.values)
	self.count = 0
	self.startTime = self.now()
	self.nextScaleTime = self.startTime.Add(edRescaleThreshold)
}

func (self *exponentiallyDecayingSample) Len() int {
	if self.count < self.reservoirSize {
		return self.count
	}
	return self.reservoirSize
}

func (self *exponentiallyDecayingSample) Values() []int64 {
	return self.values.Values()
}

func (self *exponentiallyDecayingSample) Update(value int64) {
	timestamp := self.now()
	if timestamp.After(self.nextScaleTime) {
		self.rescale(timestamp)
	}

	timestamp = self.now()
	priority := self.weight(timestamp.Sub(self.startTime)) / rand.Float64()
	self.count++
	if self.count <= self.reservoirSize {
		heap.Push(self.values, priorityValue{priority, value})
	} else {
		if first := self.values.Get(0); first.priority < priority {
			// Once Go 1.2 is release
			// self.values.samples[0] = priorityValue{priority, value}
			// heap.Fix(self.values, 0)

			heap.Pop(self.values)
			heap.Push(self.values, priorityValue{priority, value})
		}
	}
}

func (self *exponentiallyDecayingSample) weight(delta time.Duration) float64 {
	return math.Exp(self.alpha * delta.Seconds())
}

/*
A common feature of the above techniques—indeed, the key technique that
allows us to track the decayed weights efficiently—is that they maintain
counts and other quantities based on g(ti − L), and only scale by g(t − L)
at query time. But while g(ti −L)/g(t−L) is guaranteed to lie between zero
and one, the intermediate values of g(ti − L) could become very large. For
polynomial functions, these values should not grow too large, and should be
effectively represented in practice by floating point values without loss of
precision. For exponential functions, these values could grow quite large as
new values of (ti − L) become large, and potentially exceed the capacity of
common floating point types. However, since the values stored by the
algorithms are linear combinations of g values (scaled sums), they can be
rescaled relative to a new landmark. That is, by the analysis of exponential
decay in Section III-A, the choice of L does not affect the final result. We
can therefore multiply each value based on L by a factor of exp(−α(L′ − L)),
and obtain the correct value as if we had instead computed relative to a new
landmark L′ (and then use this new L′ at query time). This can be done with
a linear pass over whatever data structure is being used.
*/
func (self *exponentiallyDecayingSample) rescale(now time.Time) {
	self.nextScaleTime = now.Add(edRescaleThreshold)
	oldStartTime := self.startTime
	self.startTime = now
	scale := math.Exp(-self.alpha * self.startTime.Sub(oldStartTime).Seconds())
	self.values.ScalePriority(scale)
	self.count = self.values.Len()
}
