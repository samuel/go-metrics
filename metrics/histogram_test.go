// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

func benchmarkHistogramUpdate(b *testing.B, h Histogram) {
	for i := 0; i < b.N; i++ {
		h.Update(int64(i))
	}
}

func benchmarkHistogramPercentiles(b *testing.B, h Histogram) {
	for i := 0; i < 2000; i++ {
		h.Update(int64(i))
	}
	perc := []float64{0.5, 0.75, 0.9, 0.95, 0.99, 0.999, 0.9999}
	for i := 0; i < b.N; i++ {
		h.Percentiles(perc)
	}
}

func benchmarkHistogramConcurrentUpdate(b *testing.B, h Histogram) {
	concurrency := 100
	items := b.N / concurrency
	if items < 1 {
		items = 1
	}
	count := 0
	doneCh := make(chan bool)
	for i := 0; i < b.N; i += items {
		go func(start int) {
			for j := start; j < start+items && j < b.N; j++ {
				h.Update(int64(j))
			}
			doneCh <- true
		}(i)
		count++
	}
	for i := 0; i < count; i++ {
		_ = <-doneCh
	}
}

func TestHistogramAccuracy(t *testing.T) {
	if testing.Short() {
		return
	}

	rand.Seed(0)
	h1 := NewUnbiasedHistogram()
	h2 := NewBiasedHistogram()
	h3 := NewDefaultBucketedHistogram()
	h4 := NewDefaultMunroPatersonHistogram()
	count := 1000000
	values := int64Slice(make([]int64, count))
	for i := 0; i < count; i++ {
		v := rand.Int63n(1000000)
		h1.Update(v)
		h2.Update(v)
		h3.Update(v)
		h4.Update(v)
		values[i] = v
	}
	sort.Sort(values)
	perc := []float64{0.25, 0.5, 0.9, 0.95, 0.99, 0.999, 0.9999}
	p1 := h1.Percentiles(perc)
	p2 := h2.Percentiles(perc)
	p3 := h3.Percentiles(perc)
	p4 := h4.Percentiles(perc)
	t.Log("per\texp\tunbiased\tbiased  \tbucketed\tMP")
	for i, p := range perc {
		pos := float64(count) * p
		ipos := int(pos)
		lower := values[ipos-1]
		upper := values[ipos]
		p0 := lower + int64((pos-math.Floor(pos))*float64(upper-lower))
		e1 := 100 * math.Abs(float64(p1[i]-p0)) / float64(p0)
		e2 := 100 * math.Abs(float64(p2[i]-p0)) / float64(p0)
		e3 := 100 * math.Abs(float64(p3[i]-p0)) / float64(p0)
		e4 := 100 * math.Abs(float64(p4[i]-p0)) / float64(p0)
		t.Logf("%.4f\t%d\t%d(%.2f%%)\t%d(%.2f%%)\t%d(%.2f%%)\t%d(%.2f%%)\n",
			p, p0, p1[i], e1, p2[i], e2, p3[i], e3, p4[i], e4)
		if e1 > 5 {
			t.Errorf("Unbiased sampled histogram returned error > 5%% (%.2f%%) for percentile %.4f", e1, p)
		}
		if e2 > 5 {
			t.Errorf("Biased sampled histogram returned error > 5%% (%.2f%%) for percentile %.4f", e2, p)
		}
		if e3 > 5 {
			t.Errorf("Default bucketed histogram returned error > 5%% (%.2f%%) for percentile %.4f", e3, p)
		}
		if e4 > 5 {
			t.Errorf("Default Munro-Paterson histogram returned error > 5%% (%.2f%%) for percentile %.4f", e4, p)
		}
	}
}
