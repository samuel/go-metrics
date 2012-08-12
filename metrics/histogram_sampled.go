package metrics

import (
	"fmt"
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
	sample    Sample
	min       int64
	max       int64
	sum       int64
	count     uint64
	varianceM float64
	varianceS float64
	lock      sync.RWMutex
}

func NewSampledHistogram(sample Sample) Histogram {
	return &sampledHistogram{
		sample:    sample,
		min:       0,
		max:       0,
		sum:       0,
		count:     0,
		varianceM: 0,
		varianceS: 0}
}

/*
  Uses an exponentially decaying sample of 1028 elements, which offers
  a 99.9% confidence level with a 5% margin of error assuming a normal
  distribution, and an alpha factor of 0.015, which heavily biases
  the sample to the past 5 minutes of measurements.
*/
func NewBiasedHistogram() Histogram {
	return NewSampledHistogram(NewExponentiallyDecayingSample(1028, 0.015))
}

/*
  Uses a uniform sample of 1028 elements, which offers a 99.9%
  confidence level with a 5% margin of error assuming a normal
  distribution.
*/
func NewUnbiasedHistogram() Histogram {
	return NewSampledHistogram(NewUniformSample(1028))
}

func (h *sampledHistogram) String() string {
	return fmt.Sprintf("sampledHistogram{sum:%.4f count:%d min:%.4f max:%.4f}",
		h.sum, h.count, h.min, h.max)
}

func (h *sampledHistogram) Clear() {
	h.lock.Lock()
	h.sample.Clear()
	h.min = 0
	h.max = 0
	h.sum = 0
	h.count = 0
	h.varianceM = 0
	h.varianceS = 0
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
		h.varianceM = float64(value)
	} else {
		if value < h.min {
			h.min = value
		}
		if value > h.max {
			h.max = value
		}
		floatValue := float64(value)
		oldM := h.varianceM
		h.varianceM = oldM + ((floatValue - oldM) / float64(h.count))
		h.varianceS += (floatValue - oldM) * (floatValue - h.varianceM)
	}
	h.lock.Unlock()
}

func (h *sampledHistogram) Count() uint64 {
	return h.count
}

func (h *sampledHistogram) Sum() int64 {
	return h.sum
}

func (h *sampledHistogram) Min() int64 {
	if h.count == 0 {
		return 0
	}
	return h.min
}

func (h *sampledHistogram) Max() int64 {
	if h.count == 0 {
		return 0
	}
	return h.max
}

func (h *sampledHistogram) Mean() float64 {
	if h.count > 0 {
		return float64(h.sum) / float64(h.count)
	}
	return 0
}

func (h *sampledHistogram) StdDev() float64 {
	if h.count > 0 {
		return math.Sqrt(h.varianceS / float64(h.count-1))
	}
	return 0
}

func (h *sampledHistogram) Variance() float64 {
	if h.count <= 1 {
		return 0
	}
	return h.varianceS / float64(h.count-1)
}

func (h *sampledHistogram) Percentiles(percentiles []float64) []int64 {
	scores := make([]int64, len(percentiles))
	values := Int64Slice(h.SampleValues())
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
