// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math"
	"sort"
	"sync"
)

type Sample interface {
	Clear()
	Len() int
	Values() []int64
	Update(value int64)
}

type sampledHistogram struct {
	sample Sample
	min    int64
	max    int64
	sum    int64
	count  uint64
	lock   sync.RWMutex
}

func NewSampledHistogram(sample Sample) Histogram {
	return &sampledHistogram{sample: sample}
}

// NewBiasedHistogram returns a histogram that uses an exponentially
// decaying sample of 1028 elements, which offers
// a 99.9% confidence level with a 5% margin of error assuming a normal
// distribution, and an alpha factor of 0.015, which heavily biases
// the sample to the past 5 minutes of measurements.
func NewBiasedHistogram() Histogram {
	return NewSampledHistogram(NewExponentiallyDecayingSample(1028, 0.015))
}

// NewUnbiasedHistogram returns a histogram that uses a uniform sample
// of 1028 elements, which offers a 99.9%
// confidence level with a 5% margin of error assuming a normal
// distribution.
func NewUnbiasedHistogram() Histogram {
	return NewSampledHistogram(NewUniformSample(1028))
}

func (h *sampledHistogram) Clear() {
	h.lock.Lock()
	h.sample.Clear()
	h.min = 0
	h.max = 0
	h.sum = 0
	h.count = 0
	h.lock.Unlock()
}

func (h *sampledHistogram) Update(value int64) {
	h.lock.Lock()
	h.count++
	h.sum += value
	h.sample.Update(value)
	if h.count == 1 {
		h.min = value
		h.max = value
	} else {
		if value < h.min {
			h.min = value
		}
		if value > h.max {
			h.max = value
		}
	}
	h.lock.Unlock()
}

func (h *sampledHistogram) Count() uint64 {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.count
}

func (h *sampledHistogram) Sum() int64 {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.sum
}

func (h *sampledHistogram) Min() int64 {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if h.count == 0 {
		return 0
	}
	return h.min
}

func (h *sampledHistogram) Max() int64 {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if h.count == 0 {
		return 0
	}
	return h.max
}

func (h *sampledHistogram) Mean() float64 {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if h.count > 0 {
		return float64(h.sum) / float64(h.count)
	}
	return 0
}

func (h *sampledHistogram) Percentiles(percentiles []float64) []int64 {
	scores := make([]int64, len(percentiles))
	values := int64Slice(h.SampleValues())
	if len(values) == 0 {
		return scores
	}
	sort.Sort(values)

	for i, p := range percentiles {
		pos := p * float64(len(values)+1)
		ipos := int(pos)
		switch {
		case ipos < 1:
			scores[i] = values[0]
		case ipos >= len(values):
			scores[i] = values[len(values)-1]
		default:
			lower := values[ipos-1]
			upper := values[ipos]
			scores[i] = lower + int64((pos-math.Floor(pos))*float64(upper-lower))
		}
	}

	return scores
}

func (h *sampledHistogram) SampleValues() []int64 {
	h.lock.RLock()
	samples := h.sample.Values()
	h.lock.RUnlock()
	return samples
}

func (h *sampledHistogram) String() string {
	return histogramToJSON(h, DefaultPercentiles, DefaultPercentileNames)
}

func (h *sampledHistogram) MarshalJSON() ([]byte, error) {
	return []byte(h.String()), nil
}

func (h *sampledHistogram) MarshalText() ([]byte, error) {
	return h.MarshalJSON()
}
