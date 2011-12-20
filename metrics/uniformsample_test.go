package metrics

import (
	"testing"
)

type testUSampleStruct struct {
	reservoir_size  int
	population_size int
}

var testUSampleData = []testUSampleStruct{
	{1000, 100},
	{100, 1000},
}

func TestUSampleSizes(t *testing.T) {
	for _, data := range testUSampleData {
		sample := NewUniformSample(data.reservoir_size)
		if sample.Len() != 0 {
			t.Errorf("Size of sample should be 0 but is %d", sample.Len())
		}
		for i := 0; i < data.population_size; i++ {
			sample.Update(float64(i))
		}
		expected_size := minInt(data.reservoir_size, data.population_size)
		if sample.Len() != expected_size {
			t.Errorf("Size of sample should be %d but is %d", data.reservoir_size, sample.Len())
		}
		// Should only have elements from the population
		values := sample.GetValues()
		for i := 0; i < len(values); i++ {
			if values[i] < 0 || values[i] >= float64(data.population_size) {
				t.Errorf("Sample found that's not from population: %d", values[i])
			}
		}
	}
}
