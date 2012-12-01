// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math"
	"testing"
)

func TestBucketedHistogram(t *testing.T) {
	h := NewDefaultBucketedHistogram().(*bucketedHistogram)

	h.Update(0)
	if h.bucketCounts[0] != 1 {
		t.Fatalf("Expected 0 to fall into first bucket")
	}

	h.Clear()
	h.Update(math.MaxInt64)
	if h.bucketCounts[len(h.bucketCounts)-1] != 1 {
		t.Fatalf("Expected MaxInt64 to fall into last bucket")
	}

	h.Clear()
	h.Update(math.MinInt64)
	if h.bucketCounts[0] != 1 {
		t.Fatalf("Expected MinInt64 to fall into first bucket")
	}

	h.Clear()
	h.Update(1)
	if h.bucketCounts[1] != 1 {
		t.Fatalf("Expected 1 to fall into second bucket")
	}

	h.Clear()
	h.Update(2)
	if h.bucketCounts[2] != 1 {
		t.Fatalf("Expected 1 to fall into third bucket")
	}

	h.Clear()
	h.Update(10)
	h.Update(11)
	if h.bucketCounts[10] != 2 {
		t.Fatalf("Expected 10 & 11 to fall into 11th bucket")
	}

	h.Clear()
	h.Update(h.bucketOffsets[len(h.bucketOffsets)-1])
	if h.bucketCounts[len(h.bucketCounts)-1] != 1 {
		t.Fatal("Expected last bucket offest to fall into last count bucket")
	}

	h.Clear()
	h.Update(h.bucketOffsets[len(h.bucketOffsets)-1] + 1)
	if h.bucketCounts[len(h.bucketCounts)-1] != 1 {
		t.Fatal("Expected last bucket offest + 1 to fall into last count bucket")
	}
}

func TestBucketedHistogramPercentiles(t *testing.T) {
	h := NewDefaultBucketedHistogram().(*bucketedHistogram)

	p := h.Percentiles([]float64{0.0, 0.1, 0.5, 0.9, 1.0})
	for _, v := range p {
		if v != 0 {
			t.Fatalf("Expected empty histogram to return 0 for all percentiles instead of %d", v)
		}
	}

	h.Update(95)
	// bucket covers [91, 99], midpoint is 95
	p = h.Percentiles([]float64{0.0, 0.5, 1.0})
	if p[0] != 95 || p[1] != 95 || p[2] != 95 {
		t.Fatalf("Expected {95,95,95} instead %+v", p)
	}

	h.Clear()
	h.Update(math.MaxInt64)
	p = h.Percentiles([]float64{0.0, 0.1, 0.5, 0.9, 1.0})
	for _, v := range p {
		if v != math.MaxInt64 {
			t.Fatalf("Expected MaxInt64")
		}
	}

	tests := map[float64]int64{
		0.0:  0,
		0.5:  500,
		0.9:  900,
		0.99: 998, // 999 is a boundry
		1.0:  1000,
	}
	h.Clear()
	for i := int64(0); i < 1000; i++ {
		h.Update(i)
	}
	for perc, value := range tests {
		p := h.Percentiles([]float64{perc})
		if h.bucketIndex(p[0]) != h.bucketIndex(value) {
			t.Fatalf("%d and %d are not in the same bucket for percentile %.2f",
				p[0], value, perc)
		}
	}
}

func TestMakeBucketsForError(t *testing.T) {
	b := MakeBucketsForError(0.5)
	if len(b) != 40 {
		t.Fatalf("Number of buckets for error 0.5 should be 40 not %d", len(b))
	}
	for i, v := range []int64{1, 4, 10} {
		if b[i] != v {
			t.Fatalf("Bucket %d for error 0.5 should be %d not %d", i, v, b[i])
		}
	}
}

func BenchmarkBucketedHistogramUpdate(b *testing.B) {
	benchmarkHistogramUpdate(b, NewDefaultBucketedHistogram())
}

func BenchmarkBucketedHistogramPercentiles(b *testing.B) {
	benchmarkHistogramPercentiles(b, NewDefaultBucketedHistogram())
}

func BenchmarkBucketedHistogramConcurrentUpdate(b *testing.B) {
	benchmarkHistogramConcurrentUpdate(b, NewDefaultBucketedHistogram())
}
