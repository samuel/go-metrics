package metrics

import (
	"testing"
)

type testUSampleStruct struct {
	reservoirSize  int
	populationSize int
}

var testUSampleData = []testUSampleStruct{
	{1000, 100},
	{100, 1000},
}

func TestUSampleSizes(t *testing.T) {
	for _, data := range testUSampleData {
		sample := NewUniformSample(data.reservoirSize)
		if sample.Len() != 0 {
			t.Errorf("Size of sample should be 0 but is %d", sample.Len())
		}
		for i := 0; i < data.populationSize; i++ {
			sample.Update(float64(i))
		}
		expected_size := minInt(data.reservoirSize, data.populationSize)
		if sample.Len() != expected_size {
			t.Errorf("Size of sample should be %d but is %d", data.reservoirSize, sample.Len())
		}
		// Should only have elements from the population
		values := sample.Values()
		for i := 0; i < len(values); i++ {
			if values[i] < 0 || values[i] >= float64(data.populationSize) {
				t.Errorf("Sample found that's not from population: %d", values[i])
			}
		}
	}
}
