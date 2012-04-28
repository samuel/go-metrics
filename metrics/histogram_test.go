package metrics

import (
	"testing"
)

func TestHistogramEmpty(t *testing.T) {
	histogram := NewHistogram(NewUniformSample(100))
	if histogram.Count() != 0 {
		t.Errorf("Count for empty histogram should be 0 not %d", histogram.Count())
	}
	if histogram.Sum() != 0 {
		t.Errorf("Sum for empty histogram should be 0 not %.2f", histogram.Sum())
	}
	if histogram.Mean() != 0 {
		t.Errorf("Mean for empty histogram should be 0 not %.2f", histogram.Mean())
	}
	if histogram.StdDev() != 0 {
		t.Errorf("StdDev for empty histogram should be 0 not %.2f", histogram.StdDev())
	}
	perc := histogram.Percentiles([]float64{0.5, 0.75, 0.99})
	if len(perc) != 3 {
		t.Errorf("Percentiles expected to return slice of len 3 not %d", len(perc))
	}
	if perc[0] != 0.0 || perc[1] != 0.0 || perc[2] != 0.0 {
		t.Errorf("Percentiles returned an unexpected value (expected all 0.0)")
	}
}

func TestHistogram1to10000(t *testing.T) {
	histogram := NewHistogram(NewUniformSample(100000))
	for i := 1.0; i <= 10000; i++ {
		histogram.Update(i)
	}
	if histogram.Count() != 10000 {
		t.Errorf("Count for histogram should be 10000 not %d", histogram.Count())
	}
	if histogram.Sum() != 50005000.0 {
		t.Errorf("Sum for histogram should be 50005000 not %.2f", histogram.Sum())
	}
	if histogram.Min() != 1.0 {
		t.Errorf("Min for histogram should be 1 not %.2f", histogram.Sum())
	}
	if histogram.Max() != 10000.0 {
		t.Errorf("Max for histogram should be 10000 not %.2f", histogram.Sum())
	}
	if histogram.Mean() != 5000.5 {
		t.Errorf("Mean for histogram should be 5000.5 not %.2f", histogram.Mean())
	}
	if !almostEqual(histogram.StdDev(), 2886.896, 0.001) {
		t.Errorf("StdDev for histogram should be 2886.896 not %.3f", histogram.StdDev())
	}
	perc := histogram.Percentiles([]float64{0.5, 0.75, 0.99})
	if len(perc) != 3 {
		t.Errorf("Percentiles expected to return slice of len 3 not %d", len(perc))
	}
	if perc[0] != 5000.5 || perc[1] != 7500.75 || perc[2] != 9900.99 {
		t.Errorf("Percentiles returned an unexpected value")
	}
}
