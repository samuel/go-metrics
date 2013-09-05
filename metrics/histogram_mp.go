// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math"
	"sort"
	"sync"
)

const (
	mpElemSize = 8 // sizeof int64
)

var (
	DefaultPrecision = Precision{0.02, 100 * 1000}
	DefaultMaxMemory = 12 * 1024
)

// Precision expresses the maximum epsilon tolerated for a typical size of input
type Precision struct {
	Episilon float64
	N        int
}

type mpHistogram struct {
	buffer     [][]int64
	bufferPool [2][]int64
	indices    []int
	count      int64
	sum        int64
	min        int64
	max        int64
	leafCount  int // number of elements in the bottom two leaves
	currentTop int
	rootWeight int
	bufferSize int
	maxDepth   int
	mutex      sync.RWMutex
}

// An implemenation of the Munro-Paterson approximate histogram algorithm adapted from:
// https://github.com/twitter/commons/blob/master/src/java/com/twitter/common/stats/ApproximateHistogram.java
// http://szl.googlecode.com/svn-history/r36/trunk/src/emitters/szlquantile.cc
func NewMunroPatersonHistogram(bufSize, maxDepth int) Histogram {
	buffer := make([][]int64, maxDepth+1)
	for i := 0; i < len(buffer); i++ {
		buffer[i] = make([]int64, bufSize)
	}
	return &mpHistogram{
		buffer:     buffer,
		bufferPool: [2][]int64{make([]int64, bufSize), make([]int64, bufSize)},
		indices:    make([]int, maxDepth+1),
		rootWeight: 1,
		currentTop: 1,
		bufferSize: bufSize,
		maxDepth:   maxDepth,
	}
}

func NewDefaultMunroPatersonHistogram() Histogram {
	return NewMunroPatersonHistogramWithMaxMemory(DefaultMaxMemory)
}

func NewMunroPatersonHistogramWithMaxMemory(bytes int) Histogram {
	depth := computeDepth(DefaultPrecision.Episilon, DefaultPrecision.N)
	bufSize := computeBufferSize(depth, DefaultPrecision.N)
	maxDepth := computeMaxDepth(bytes, bufSize)
	return NewMunroPatersonHistogram(bufSize, maxDepth)
}

func NewMunroPatersonHistogramWithPrecision(p Precision) Histogram {
	depth := computeDepth(p.Episilon, p.N)
	bufSize := computeBufferSize(depth, p.N)
	return NewMunroPatersonHistogram(bufSize, depth)
}

func (mp *mpHistogram) String() string {
	return histogramToJson(mp, DefaultPercentiles, DefaultPercentileNames)
}

func (mp *mpHistogram) Clear() {
	mp.mutex.Lock()
	mp.count = 0
	mp.sum = 0
	mp.leafCount = 0
	mp.rootWeight = 1
	mp.min = 0
	mp.max = 0
	mp.mutex.Unlock()
}

func (mp *mpHistogram) Count() uint64 {
	mp.mutex.RLock()
	count := uint64(mp.count)
	mp.mutex.RUnlock()
	return count
}

func (mp *mpHistogram) Mean() float64 {
	mp.mutex.RLock()
	mean := float64(mp.sum) / float64(mp.count)
	mp.mutex.RUnlock()
	return mean
}

func (mp *mpHistogram) Sum() int64 {
	mp.mutex.RLock()
	sum := mp.sum
	mp.mutex.RUnlock()
	return sum
}

func (mp *mpHistogram) Min() int64 {
	mp.mutex.RLock()
	min := mp.min
	mp.mutex.RUnlock()
	return min
}

func (mp *mpHistogram) Max() int64 {
	mp.mutex.RLock()
	max := mp.max
	mp.mutex.RUnlock()
	return max
}

func (mp *mpHistogram) Percentiles(qs []float64) []int64 {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	output := make([]int64, len(qs))
	if mp.count == 0 {
		return output
	}

	// the two leaves are the only buffer that can be partially filled
	buf0Size := mp.leafCount
	buf1Size := 0
	if mp.leafCount > mp.bufferSize {
		buf0Size = mp.bufferSize
		buf1Size = mp.leafCount - mp.bufferSize
	}

	sort.Sort(int64Slice(mp.buffer[0][:buf0Size]))
	sort.Sort(int64Slice(mp.buffer[1][:buf1Size]))

	indices := mp.indices
	for i := 0; i < len(indices); i++ {
		indices[i] = 0
	}
	sum := int64(0)
	io := 0
	floatCount := float64(mp.count)
	for io < len(output) {
		i := mp.smallest(buf0Size, buf1Size, indices)
		id := indices[i]
		indices[i]++
		sum += int64(mp.weight(i))
		for io < len(qs) && int64(qs[io]*floatCount) <= sum {
			output[io] = mp.buffer[i][id]
			io++
		}
	}
	return output
}

// Return the level of the smallest element (using the indices array 'ids'
// to track which elements have been already returned). Every buffers has
// already been sorted at this point.
func (mp *mpHistogram) smallest(buf0Size, buf1Size int, ids []int) int {
	smallest := int64(math.MaxInt64)
	id0 := ids[0]
	id1 := ids[1]
	iSmallest := 0

	if mp.leafCount > 0 && id0 < buf0Size {
		smallest = mp.buffer[0][id0]
	}
	if mp.leafCount > mp.bufferSize && id1 < buf1Size {
		x := mp.buffer[1][id1]
		if x < smallest {
			smallest = x
			iSmallest = 1
		}
	}
	for i := 2; i <= mp.currentTop; i++ {
		if !mp.isBufferEmpty(i) && ids[i] < mp.bufferSize {
			x := mp.buffer[i][ids[i]]
			if x < smallest {
				smallest = x
				iSmallest = i
			}
		}
	}
	return iSmallest
}

func (mp *mpHistogram) Update(x int64) {
	mp.mutex.Lock()
	// if the leaves of the tree are full, "collapse" recursively the tree
	if mp.leafCount == 2*mp.bufferSize {
		sort.Sort(int64Slice(mp.buffer[0]))
		sort.Sort(int64Slice(mp.buffer[1]))
		mp.recCollapse(mp.buffer[0], 1)
		mp.leafCount = 0
	}

	// Now we're sure there is space for adding x
	if mp.leafCount < mp.bufferSize {
		mp.buffer[0][mp.leafCount] = x
	} else {
		mp.buffer[1][mp.leafCount-mp.bufferSize] = x
	}
	mp.leafCount++
	if mp.count == 0 {
		mp.min = x
		mp.max = x
	} else {
		if x < mp.min {
			mp.min = x
		}
		if x > mp.max {
			mp.max = x
		}
	}
	mp.count++
	mp.sum += x
	mp.mutex.Unlock()
}

func (mp *mpHistogram) recCollapse(buf []int64, level int) {
	// if we reach the root, we can't add more buffer
	if level == mp.maxDepth {
		// weight() returns the weight of the root, in that case we need the
		// weight of merge result
		mergeWeight := 1 << (uint(level) - 1)
		idx := level & 1
		merged := mp.bufferPool[idx]
		tmp := mp.buffer[level]
		if mergeWeight == mp.rootWeight {
			mp.collapse1(buf, mp.buffer[level], merged)
		} else {
			mp.collapse(buf, mergeWeight, mp.buffer[level], mp.rootWeight, merged)
		}
		mp.buffer[level] = merged
		mp.bufferPool[idx] = tmp
		mp.rootWeight += mergeWeight
	} else {
		if level == mp.currentTop {
			// if we reach the top, add a new buffer
			mp.collapse1(buf, mp.buffer[level], mp.buffer[level+1])
			mp.currentTop++
			mp.rootWeight *= 2
		} else if mp.isBufferEmpty(level + 1) {
			// if the upper buffer is empty, use it
			mp.collapse1(buf, mp.buffer[level], mp.buffer[level+1])
		} else {
			// if the upper buffer isn't empty, collapse with it
			merged := mp.bufferPool[level&1]
			mp.collapse1(buf, mp.buffer[level], merged)
			mp.recCollapse(merged, level+1)
		}
	}
}

// collapse two sorted Arrays of different weight
// ex: [2,5,7] weight 2 and [3,8,9] weight 3
//     weight x array + concat = [2,2,5,5,7,7,3,3,3,8,8,8,9,9,9]
//     sort = [2,2,3,3,3,5,5,7,7,8,8,8,9,9,9]
//     select every nth elems = [3,7,9]  (n = sum weight / 2)
func (mp *mpHistogram) collapse(left []int64, leftWeight int, right []int64, rightWeight int, output []int64) {
	totalWeight := leftWeight + rightWeight
	i := 0
	j := 0
	k := 0
	cnt := 0

	var smallest int64
	var weight int

	for i < len(left) || j < len(right) {
		if i < len(left) && (j == len(right) || left[i] < right[j]) {
			smallest = left[i]
			weight = leftWeight
			i++
		} else {
			smallest = right[j]
			weight = rightWeight
			j++
		}

		cur := (cnt + (totalWeight >> 1) - 1) / totalWeight
		cnt += weight
		next := (cnt + (totalWeight >> 1) - 1) / totalWeight

		for ; cur < next; cur++ {
			output[k] = smallest
			k++
		}
	}
}

// Optimized version of collapse for collapsing two arrays of the
// same weight (which is what we want most of the time)
func (mp *mpHistogram) collapse1(left, right, output []int64) {
	i, j, k, cnt := 0, 0, 0, 0
	ll := len(left)
	lr := len(right)
	for i < ll || j < lr {
		var smallest int64
		if i < ll && (j == lr || left[i] < right[j]) {
			smallest = left[i]
			i++
		} else {
			smallest = right[j]
			j++
		}
		if cnt&1 == 1 {
			output[k] = smallest
			k++
		}
		cnt++
	}
}

func (mp *mpHistogram) isBufferEmpty(level int) bool {
	if level == mp.currentTop {
		return false // root buffer (is present) is always full
	}
	return (mp.count/int64(mp.bufferSize*mp.weight(level)))&1 == 1
}

// return the weight of the level ie. 2^(i-1) except for the two tree
// leaves (weight=1) and for the root
func (mp *mpHistogram) weight(level int) int {
	if level < 2 {
		return 1
	}
	if level == mp.maxDepth {
		return mp.rootWeight
	}
	return 1 << (uint(level) - 1)
}

//

// We compute the "smallest possible k" satisfying two inequalities:
//    1)   (b - 2) * (2 ^ (b - 2)) + 0.5 <= epsilon * N
//    2)   k * (2 ^ (b - 1)) >= N
//
// For an explanation of these inequalities, please read the Munro-Paterson or
// the Manku-Rajagopalan-Linday papers.
func computeDepth(epsilon float64, n int) int {
	b := uint(2)
	en := epsilon * float64(n)
	for float64((b-2)*(1<<(b-2)))+0.5 <= en {
		b++
	}
	return int(b)
}

func computeBufferSize(b int, n int) int {
	return int(n / (1 << (uint(b) - 1)))
}

// Return the maximum depth of the graph to comply with the memory constraint
func computeMaxDepth(maxMemoryBytes int, bufferSize int) int {
	bm := 0
	n := maxMemoryBytes - 100 - mpElemSize*bufferSize
	if n < 0 {
		bm = 2
	} else {
		bm = int(n / (16 + mpElemSize*bufferSize))
	}
	if bm < 2 {
		bm = 2
	}
	return bm
}
