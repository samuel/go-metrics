// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"testing"
	"time"
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
		expectedSize := minInt(data.reservoirSize, data.populationSize)
		if sample.Len() != expectedSize {
			t.Errorf("Size of sample should be %d but is %d", data.reservoirSize, sample.Len())
		}
		// Should only have elements from the population
		if val, ok := allValuesBetween(sample.Values(), 0, int64(data.populationSize)); !ok {
			t.Errorf("Sample found that's not from population: %d", val)
		}
	}
}

func allValuesBetween(values []int64, min, max int64) (int64, bool) {
	for _, v := range values {
		if v < min || v > max {
			return v, false
		}
	}
	return 0, true
}

// long periods of inactivity should not corrupt sampling state
func TestEDSampleInactivity(t *testing.T) {
	curTime := time.Time{}
	timefunc := func() time.Time {
		return curTime
	}

	sample := NewExponentiallyDecayingSampleWithCustomTime(10, 0.015, timefunc)
	realSample := sample.(*exponentiallyDecayingSample)

	// add 1000 values at a rate of 10 values/second
	for i := int64(0); i < 1000; i++ {
		sample.Update(1000 + i)
		curTime = curTime.Add(time.Millisecond * 100)
	}
	if len(sample.Values()) != 10 {
		t.Fatalf("Expected 10 samples instead of %d", len(sample.Values()))
	}
	if val, ok := allValuesBetween(sample.Values(), 1000, 2000); !ok {
		t.Errorf("Sample found that's not from population: %d", val)
	}

	// wait for 15 hours and add another value.
	// this should trigger a rescale. Note that the number of samples will be reduced to 2
	// because of the very small scaling factor that will make all existing priorities equal to
	// zero after rescale.
	curTime = curTime.Add(time.Hour * 15)
	sample.Update(2000)
	if val, ok := allValuesBetween(sample.Values(), 1000, 3000); !ok {
		t.Errorf("Sample found that's not from population: %d", val)
	}
	// if len(sample.Values()) != 2 {
	// 	t.Fatalf("Expected 2 samples instead of %d: %+v", len(sample.Values()), realSample.values)
	// }

	// add 1000 values at a rate of 10 values/second
	for i := int64(0); i < 1000; i++ {
		sample.Update(3000 + i)
		curTime = curTime.Add(time.Millisecond * 100)
	}
	if len(sample.Values()) != 10 {
		t.Fatalf("Expected 10 samples instead of %d: %+v", len(sample.Values()), realSample.values)
	}
	if val, ok := allValuesBetween(sample.Values(), 3000, 4000); !ok {
		t.Errorf("Sample found that's not from population: %d", val)
	}
}

func BenchmarkEDSampleUpdate(b *testing.B) {
	sample := NewExponentiallyDecayingSample(1000, 0.99)
	for i := 0; i < b.N; i++ {
		sample.Update(int64(i))
	}
}
