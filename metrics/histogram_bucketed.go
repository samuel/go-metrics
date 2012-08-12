package metrics

import (
	"math"
	"sync"
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
	varianceM     float64
	varianceS     float64
	lock          sync.RWMutex
}

// private[this] def makeBucketsFor(error: Double): Array[Int] = {
//   def build(factor: Double, n: Double): Stream[Double] = {
//     val next = n * factor
//     if (next.toInt == Int.MaxValue) Stream.empty else Stream.cons(next, build(factor, next))
//   }

//   val factor = (1.0 + error) / (1.0 - error)
//   (Seq(1) ++ build(factor, 1.0).map(_.toInt + 1).distinct.force).toArray
// }

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

// Implements https://github.com/twitter/ostrich/blob/master/src/main/scala/com/twitter/ostrich/stats/Histogram.scala
func NewBucketedHistogram(bucketOffsets []int64) Histogram {
	return &bucketedHistogram{
		bucketOffsets: bucketOffsets,
		bucketCounts:  make([]uint64, len(bucketOffsets)+1),
	}
}

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
	h.lock.Lock()
	h.min = 0
	h.max = 0
	h.sum = 0
	h.count = 0
	h.varianceM = 0
	h.varianceS = 0
	for i := 0; i < len(h.bucketCounts); i++ {
		h.bucketCounts[i] = 0
	}
	h.lock.Unlock()
}

func (h *bucketedHistogram) Update(value int64) {
	h.lock.Lock()
	bucketIndex := h.bucketIndex(value)
	h.bucketCounts[bucketIndex]++
	h.count++
	h.sum += value
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
		floatValue := float64(value)
		oldM := h.varianceM
		h.varianceM = oldM + ((floatValue - oldM) / float64(h.count))
		h.varianceS += (floatValue - oldM) * (floatValue - h.varianceM)
	}
	h.lock.Unlock()
}

func (h *bucketedHistogram) Count() uint64 {
	return h.count
}

func (h *bucketedHistogram) Sum() int64 {
	return h.sum
}

func (h *bucketedHistogram) Min() int64 {
	if h.count == 0 {
		return 0
	}
	return h.min
}

func (h *bucketedHistogram) Max() int64 {
	if h.count == 0 {
		return 0
	}
	return h.max
}

func (h *bucketedHistogram) Mean() float64 {
	if h.count > 0 {
		return float64(h.sum) / float64(h.count)
	}
	return 0
}

func (h *bucketedHistogram) StdDev() float64 {
	if h.count > 0 {
		return math.Sqrt(h.varianceS / float64(h.count-1))
	}
	return 0
}

func (h *bucketedHistogram) Variance() float64 {
	if h.count <= 1 {
		return 0
	}
	return h.varianceS / float64(h.count-1)
}

func (h *bucketedHistogram) Percentiles(percentiles []float64) []int64 {
	scores := make([]int64, len(percentiles))

	total := uint64(0)
	index := 0
	for i, p := range percentiles {
		if p == 0.0 {
			scores[i] = h.min
		} else {
			target := p * float64(h.count)
			for float64(total) < target {
				total += h.bucketCounts[index]
				index++
			}
			if index <= 1 {
				scores[i] = 0
			} else if index-1 >= len(h.bucketOffsets) {
				scores[i] = math.MaxInt64
			} else {
				scores[i] = (h.bucketOffsets[index-2] + h.bucketOffsets[index-1] - 1) >> 1
			}
		}
	}

	return scores
}
