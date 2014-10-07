// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math"
	"sync"
	"sync/atomic"
)

var (
	bucketCache     = make(map[float64][]int64, 0) // cache of buckets for an error rate
	bucketCacheLock sync.Mutex
)

type bucketedHistogram struct {
	bucketOffsets []int64
	bucketCounts  []uint64
	min           int64
	max           int64
	sum           int64
	count         uint64
	mu            sync.RWMutex
}

// MakeBucketsForError compute all the bucket values from 1 until we run out of positive 64-bit ints
// for the given an error (+/-). The error should be in percent, between 0.0 and 1.0.
//
// Each bucket's value will be the midpoint of an error range to the edge of the bucket in each
// direction, so for example, given a 5% error range (the default), the bucket with value N will
// cover numbers 5% smaller (0.95*N) and 5% larger (1.05*N).
//
// For the usual default of 5%, this results in 200 buckets.
//
// The last bucket (the "infinity" bucket) ranges up to math.MaxInt64, which we treat as infinity.
func MakeBucketsForError(error float64) []int64 {
	bucketCacheLock.Lock()
	defer bucketCacheLock.Unlock()

	bucketOffsets := bucketCache[error]
	if bucketOffsets == nil {
		bucketOffsets = make([]int64, 1)
		bucketOffsets[0] = 1
		lastValue := int64(1)
		factor := (1.0 + error) / (1.0 - error)
		max := float64(math.MaxInt64)
		next := 1.0
		for {
			next = next * factor
			if next >= max {
				break
			} else {
				value := int64(next) + 1
				if value != lastValue {
					bucketOffsets = append(bucketOffsets, value)
					lastValue = value
				}
			}
		}

		bucketCache[error] = bucketOffsets
	}
	return bucketOffsets
}

// NewBucketedHistogram returns a histogram that uses a fixed set of buckets for ranges of values.
// This is an implementation of the Histogram class from Ostrich.
// https://github.com/twitter/ostrich/blob/master/src/main/scala/com/twitter/ostrich/stats/Histogram.scala
func NewBucketedHistogram(bucketOffsets []int64) Histogram {
	return &bucketedHistogram{
		bucketOffsets: bucketOffsets,
		bucketCounts:  make([]uint64, len(bucketOffsets)+1),
		min:           math.MaxInt64,
		max:           math.MinInt64,
	}
}

// NewDefaultBucketedHistogram returns a bucketed histogram with an error of 5%
func NewDefaultBucketedHistogram() Histogram {
	return NewBucketedHistogram(MakeBucketsForError(0.05))
}

func (h *bucketedHistogram) bucketIndex(key int64) int {
	low := 0
	high := len(h.bucketOffsets) - 1
	for low <= high {
		mid := (low + high + 1) >> 1
		midValue := h.bucketOffsets[mid]
		if midValue < key {
			low = mid + 1
		} else if midValue > key {
			high = mid - 1
		} else {
			// exactly equal to this bucket's value. but the value is an exclusive max, so bump it up.
			return mid + 1
		}
	}
	return low
}

func (h *bucketedHistogram) Clear() {
	h.mu.Lock()
	h.count = 0
	h.sum = 0
	h.min = math.MaxInt64
	h.max = math.MinInt64
	for i := 0; i < len(h.bucketCounts); i++ {
		h.bucketCounts[i] = 0
	}
	h.mu.Unlock()
}

func (h *bucketedHistogram) Update(value int64) {
	h.mu.Lock()
	bucketIndex := h.bucketIndex(value)
	h.bucketCounts[bucketIndex] += 1
	h.count++
	atomic.AddInt64(&h.sum, value)
	h.sum += value
	if value < h.min {
		h.min = value
	}
	if value > h.max {
		h.max = value
	}
	h.mu.Unlock()
}

func (h *bucketedHistogram) Distribution() DistributionValue {
	h.mu.RLock()
	v := DistributionValue{
		Count: h.count,
		Sum:   float64(h.sum),
	}
	if h.count > 0 {
		v.Min = float64(h.min)
		v.Max = float64(h.max)
	}
	h.mu.RUnlock()
	return v
}

func (h *bucketedHistogram) Percentiles(percentiles []float64) []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	scores := make([]int64, len(percentiles))

	total := uint64(0)
	index := 0
	for i, p := range percentiles {
		if p > 1.0 {
			p /= 100.0
		}
		if p == 0.0 {
			if h.count == 0 {
				scores[i] = 0
			} else {
				scores[i] = h.min
			}
		} else {
			target := p * float64(h.count)
			for float64(total) < target {
				total += atomic.LoadUint64(&h.bucketCounts[index])
				index++
			}
			if index <= 1 {
				scores[i] = 0
			} else if index-1 >= len(h.bucketOffsets) {
				scores[i] = math.MaxInt64
			} else {
				// Avoid overflow calculating (h.bucketOffsets[index-2] + h.bucketOffsets[index-1] - 1) >> 1
				o1 := h.bucketOffsets[index-2]
				o2 := h.bucketOffsets[index-1]
				bit := ((o1 & 1) | (o1 & 1)) ^ 1
				scores[i] = (o1 >> 1) + (o2 >> 1) - bit
			}
		}
	}

	return scores
}

func (h *bucketedHistogram) String() string {
	return histogramToJSON(h, DefaultPercentiles, DefaultPercentileNames)
}

func (h *bucketedHistogram) MarshalJSON() ([]byte, error) {
	return []byte(h.String()), nil
}

func (h *bucketedHistogram) MarshalText() ([]byte, error) {
	return h.MarshalJSON()
}
