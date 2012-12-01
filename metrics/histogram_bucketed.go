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
	lock          sync.RWMutex
}

// Given an error (+/-), compute all the bucket values from 1 until we run out of positive
// 64-bit ints. The error should be in percent, between 0.0 and 1.0.
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

// A histogram that uses a fixed set of buckets for ranges of values.
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

// Create a bucketed histogram with an error of 5%
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
	atomic.StoreUint64(&h.count, 0)
	atomic.StoreInt64(&h.sum, 0)
	atomic.StoreInt64(&h.min, math.MaxInt64)
	atomic.StoreInt64(&h.max, math.MinInt64)
	for i := 0; i < len(h.bucketCounts); i++ {
		atomic.StoreUint64(&h.bucketCounts[i], 0)
	}
}

func (h *bucketedHistogram) Update(value int64) {
	bucketIndex := h.bucketIndex(value)
	atomic.AddUint64(&h.bucketCounts[bucketIndex], 1)
	atomic.AddUint64(&h.count, 1)
	atomic.AddInt64(&h.sum, value)
	for {
		min := atomic.LoadInt64(&h.min)
		if value > min || atomic.CompareAndSwapInt64(&h.min, min, value) {
			break
		}
	}
	for {
		max := atomic.LoadInt64(&h.max)
		if value < max || atomic.CompareAndSwapInt64(&h.max, max, value) {
			break
		}
	}
}

func (h *bucketedHistogram) Count() uint64 {
	return atomic.LoadUint64(&h.count)
}

func (h *bucketedHistogram) Sum() int64 {
	return atomic.LoadInt64(&h.sum)
}

func (h *bucketedHistogram) Min() int64 {
	return atomic.LoadInt64(&h.min)
}

func (h *bucketedHistogram) Max() int64 {
	return atomic.LoadInt64(&h.max)
}

func (h *bucketedHistogram) Mean() float64 {
	count := h.Count()
	if count > 0 {
		return float64(h.Sum()) / float64(count)
	}
	return 0
}

func (h *bucketedHistogram) Percentiles(percentiles []float64) []int64 {
	scores := make([]int64, len(percentiles))

	total := uint64(0)
	index := 0
	for i, p := range percentiles {
		if p == 0.0 {
			if h.Count() == 0 {
				scores[i] = 0
			} else {
				scores[i] = h.Min()
			}
		} else {
			target := p * float64(h.Count())
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
	return histogramToJson(h, DefaultPercentiles, DefaultPercentileNames)
}
