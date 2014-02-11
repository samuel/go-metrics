// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

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
			sample.Update(int64(i))
		}
		expectedSize := minInt(data.reservoirSize, data.populationSize)
		if sample.Len() != expectedSize {
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

func BenchmarkUniformSampleUpdate(b *testing.B) {
	sample := NewUniformSample(1000)
	for i := 0; i < b.N; i++ {
		sample.Update(int64(i))
	}
}
