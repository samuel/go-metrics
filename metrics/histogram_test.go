package metrics

import "testing"

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
