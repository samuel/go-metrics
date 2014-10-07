// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"testing"
)

func TestSampledHistogramEmpty(t *testing.T) {
	histogram := NewSampledHistogram(NewUniformSample(100))
	v := histogram.Distribution()
	if v.Count != 0 {
		t.Errorf("Count for empty histogram should be 0 not %d", v.Count)
	}
	if v.Sum != 0 {
		t.Errorf("Sum for empty histogram should be 0 not %f", v.Sum)
	}
	perc := histogram.Percentiles([]float64{0.5, 0.75, 0.99})
	if len(perc) != 3 {
		t.Errorf("Percentiles expected to return slice of len 3 not %d", len(perc))
	}
	if perc[0] != 0.0 || perc[1] != 0.0 || perc[2] != 0.0 {
		t.Errorf("Percentiles returned an unexpected value (expected all 0.0)")
	}
}

func TestSampledHistogram1to10000(t *testing.T) {
	histogram := NewSampledHistogram(NewUniformSample(100000))
	for i := int64(1); i <= 10000; i++ {
		histogram.Update(i)
	}
	v := histogram.Distribution()
	if v.Count != 10000 {
		t.Errorf("Count for histogram should be 10000 not %d", v.Count)
	}
	if v.Sum != 50005000 {
		t.Errorf("Sum for histogram should be 50005000 not %f", v.Sum)
	}
	if v.Min != 1 {
		t.Errorf("Min for histogram should be 1 not %f", v.Min)
	}
	if v.Max != 10000 {
		t.Errorf("Max for histogram should be 10000 not %f", v.Max)
	}
	perc := histogram.Percentiles([]float64{0.5, 0.75, 0.99})
	if len(perc) != 3 {
		t.Errorf("Percentiles expected to return slice of len 3 not %d", len(perc))
	}
	if perc[0] != 5000 || perc[1] != 7500 || perc[2] != 9900 {
		t.Errorf("Percentiles returned an unexpected value")
	}
}

func BenchmarkUniformSampledHistogramUpdate(b *testing.B) {
	benchmarkHistogramUpdate(b, NewSampledHistogram(NewUniformSample(1028)))
}

func BenchmarkUniformSampledHistogramPercentiles(b *testing.B) {
	benchmarkHistogramPercentiles(b, NewSampledHistogram(NewUniformSample(1028)))
}

func BenchmarkBiasedSampledHistogramUpdate(b *testing.B) {
	benchmarkHistogramUpdate(b, NewBiasedHistogram())
}

func BenchmarkBiasedSampledHistogramPercentiles(b *testing.B) {
	benchmarkHistogramPercentiles(b, NewBiasedHistogram())
}
