package metrics

import (
	"testing"
)

type testEDSampleStruct struct {
	reservoirSize  int
	populationSize int
	alpha          float64
}

var testEDData = []testEDSampleStruct{
	{1000, 100, 0.99},
	{100, 1000, 0.99},
	{1000, 100, 0.01},
}

func TestEDSampleSizes(t *testing.T) {
	for _, data := range testEDData {
		sample := NewExponentiallyDecayingSample(data.reservoirSize, data.alpha)
		if sample.Len() != 0 {
			t.Errorf("Size of sample should be 0 but is %d", sample.Len())
		}
		for i := 0; i < data.populationSize; i++ {
			sample.Update(int64(i))
		}
		expected_size := minInt(data.reservoirSize, data.populationSize)
		if sample.Len() != expected_size {
			t.Errorf("Size of sample should be %d but is %d", data.reservoirSize, sample.Len())
		}
		// Should only have elements from the population
		values := sample.Values()
		for i := 0; i < len(values); i++ {
			if values[i] < 0 || values[i] >= int64(data.populationSize) {
				t.Errorf("Sample found that's not from population: %d", values[i])
			}
		}
	}
}

func BenchmarkEDSampleUpdate(b *testing.B) {
	sample := NewExponentiallyDecayingSample(1000, 0.99)
	for i := 0; i < b.N; i++ {
		sample.Update(int64(i))
	}
}
