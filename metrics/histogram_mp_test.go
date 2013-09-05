// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestMP(t *testing.T) {
	hist := NewDefaultMunroPatersonHistogram().(*mpHistogram)

	for i := 0; i < hist.bufferSize*4; i++ {
		hist.Update(int64(i + 1))
	}
	hist.Update(int64(1))
	if hist.currentTop != 3 {
		t.Fatalf("mp.currentTop is %d instead of 3", hist.currentTop)
	}

	for i := 0; i <= hist.currentTop; i++ {
		for j := 0; j < hist.bufferSize; j++ {
			if hist.buffer[i][j] == 0 {
				t.Fatalf("mp.buffer[%d][%d] == 0", i, j)
			}
		}
	}
}

func TestMPCollapse(t *testing.T) {
	hist := NewDefaultMunroPatersonHistogram().(*mpHistogram)
	buf1 := []int64{2, 5, 7}
	buf2 := []int64{3, 8, 9}
	expected := []int64{3, 7, 9}
	result := make([]int64, 3)

	// [2,5,7] weight 2 and [3,8,9] weight 3
	// weight x array + concat = [2,2,5,5,7,7,3,3,3,8,8,8,9,9,9]
	// sort = [2,2,3,3,3,5,5,7,7,8,8,8,9,9,9]
	// select every nth elems = [3,7,9]  (n = sum weight / 2, ie. 5/3 = 2)
	// [2,2,3,3,3,5,5,7,7,8,8,8,9,9,9]
	//  . . ^ . . . . ^ . . . . ^ . .
	//  [-------] [-------] [-------] we make 3 packets of 5 elements and take the middle

	hist.collapse(buf1, 2, buf2, 3, result)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("mp.combine returned %+v instead of %+v", result, expected)
	}

	buf1 = []int64{2, 5, 7, 9}
	buf2 = []int64{3, 8, 9, 12}
	expected = []int64{3, 7, 9, 12}
	result = make([]int64, 4)

	hist.collapse(buf1, 2, buf2, 2, result)
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("mp.combine returned %+v instead of %+v", result, expected)
	}

	rand.Seed(0)
	buf1 = make([]int64, 15625)
	buf2 = make([]int64, 15625)
	result = make([]int64, 15625)
	for i := 0; i < len(buf1); i++ {
		buf1[i] = rand.Int63()
		buf2[i] = rand.Int63()
	}
	hist.collapse(buf1, 2, buf2, 10, result)
	n := 0
	for i := 0; i < len(result); i++ {
		if result[i] == 0 {
			break
		}
		n++
	}
	if n != len(result) {
		t.Fatalf("mp.combine only filled result to %d instead of %d for weights 2, 8", n, len(result))
	}
}

func TestMPRecCollapse(t *testing.T) {
	b := 10
	n := 3
	empty := []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	full := []int64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	hist := NewMunroPatersonHistogram(b, n).(*mpHistogram)

	if !reflect.DeepEqual(empty, hist.buffer[0]) {
		t.Fatalf("mp.buffer[0] should be empty")
	}
	if !reflect.DeepEqual(empty, hist.buffer[1]) {
		t.Fatalf("mp.buffer[1] should be empty")
	}

	addToHist(hist, b)

	if !reflect.DeepEqual(full, hist.buffer[0]) {
		t.Fatalf("mp.buffer[0] should be full")
	}
	if !reflect.DeepEqual(empty, hist.buffer[1]) {
		t.Fatalf("mp.buffer[1] should be empty")
	}

	addToHist(hist, b)

	if !reflect.DeepEqual(full, hist.buffer[0]) {
		t.Fatalf("mp.buffer[0] should be full")
	}
	if !reflect.DeepEqual(full, hist.buffer[1]) {
		t.Fatalf("mp.buffer[1] should be full")
	}

	hist.Update(1)

	if hist.currentTop != 2 {
		t.Fatalf("mp.currentTop is %d instead of 2", hist.currentTop)
	}
	// Buffers are not cleared so we can't check that!
	if !reflect.DeepEqual(full, hist.buffer[2]) {
		t.Fatalf("mp.buffer[2] should be full")
	}

	addToHist(hist, 2*b)

	if hist.currentTop != 3 {
		t.Fatalf("mp.currentTop is %d instead of 3", hist.currentTop)
	}
	if !reflect.DeepEqual(full, hist.buffer[3]) {
		t.Fatalf("mp.buffer[3] should be full")
	}
}

func TestMPReachingMaxDepth(t *testing.T) {
	b := 10
	h := 3
	hist := NewMunroPatersonHistogram(b, h).(*mpHistogram)

	addToHist(hist, 8*b)
	if hist.currentTop != 3 {
		t.Fatalf("mp.currentTop is %d instead of 3", hist.currentTop)
	}

	hist.Update(1)
	if hist.currentTop != 3 {
		t.Fatalf("mp.currentTop is %d instead of 3", hist.currentTop)
	}
}

func TestMPSmallestIndexFinder(t *testing.T) {
	b := 10
	h := 3
	hist := NewMunroPatersonHistogram(b, h).(*mpHistogram)

	for i := 1; i <= 3; i++ {
		hist.Update(int64(i))
	}
	for i := 1; i <= 3; i++ {
		j := hist.smallest(3, 0, hist.indices)
		idx := hist.indices[j]
		hist.indices[j] += 1
		if int64(i) != hist.buffer[j][idx] {
			t.Fatalf("mp.smallest failed 1")
		}
	}

	for i := 0; i < len(hist.indices); i++ {
		hist.indices[i] = 0
	}
	for i := 4; i <= 2*b; i++ {
		hist.Update(int64(i))
	}
	for i := 1; i <= 2*b; i++ {
		j := hist.smallest(b, b, hist.indices)
		idx := hist.indices[j]
		hist.indices[j] += 1
		if int64(i) != hist.buffer[j][idx] {
			t.Fatalf("mp.smallest failed 2")
		}
	}

	for i := 0; i < len(hist.indices); i++ {
		hist.indices[i] = 0
	}
	hist.Update(int64(2*b + 1))
	for i := 2; i <= 2*b+1; i += 2 {
		j := hist.smallest(1, 0, hist.indices)
		idx := hist.indices[j]
		hist.indices[j] += 1
		if int64(i) != hist.buffer[j][idx] {
			t.Fatalf("mp.smallest failed 3")
		}
	}

	j := hist.smallest(1, 0, hist.indices)
	idx := hist.indices[j]
	hist.indices[j] += 1
	if int64(2*b+1) != hist.buffer[j][idx] {
		t.Fatalf("mp.smallest failed 4")
	}
}

func addToHist(hist Histogram, n int) {
	for i := 0; i < n; i++ {
		hist.Update(1)
	}
}

func BenchmarkMPDefaultHistogramUpdate(b *testing.B) {
	benchmarkHistogramUpdate(b, NewDefaultMunroPatersonHistogram())
}

func BenchmarkMPDefaultHistogramPercentiles(b *testing.B) {
	benchmarkHistogramPercentiles(b, NewDefaultMunroPatersonHistogram())
}

func BenchmarkMPDefaultHistogramConcurrentUpdate(b *testing.B) {
	benchmarkHistogramConcurrentUpdate(b, NewDefaultMunroPatersonHistogram())
}
