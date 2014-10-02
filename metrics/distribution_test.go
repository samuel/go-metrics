// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import "testing"

func TestDistribution(t *testing.T) {
	d := NewDistribution()
	if v := d.Min(); v != 0.0 {
		t.Errorf("Expected new distribution to have min 0.0. Got %f", v)
	}
	if v := d.Max(); v != 0.0 {
		t.Errorf("Expected new distribution to have max 0.0. Got %f", v)
	}
	d.Update(2.0)
	d.Update(9.0)
	d.Update(4.0)
	if v := d.Min(); v != 2.0 {
		t.Errorf("Expected min of 2.0. Got %f", v)
	}
	if v := d.Max(); v != 9.0 {
		t.Errorf("Expected max of 9.0. Got %f", v)
	}
	if v := d.Count(); v != 3 {
		t.Errorf("Expected count of 3. Got %d", v)
	}
	if v := d.Sum(); v != 15.0 {
		t.Errorf("Expected sum of 15.0. Got %f", v)
	}
	if v := d.Variance(); v != 13.0 {
		t.Errorf("Expected variance of 13.0. Got %f", v)
	}
	v := d.Value()
	if v.Min != 2.0 {
		t.Errorf("Expected min of 2.0. Got %f", v.Min)
	}
	if v.Max != 9.0 {
		t.Errorf("Expected max of 9.0. Got %f", v.Max)
	}
	if v.Count != 3 {
		t.Errorf("Expected count of 3. Got %d", v.Count)
	}
	if v.Sum != 15.0 {
		t.Errorf("Expected sum of 15.0. Got %f", v.Sum)
	}
	if v.Variance != 13.0 {
		t.Errorf("Expected variance of 13.0. Got %f", v.Variance)
	}
}

func BenchmarkDistributionConcurrentUpdate(b *testing.B) {
	concurrency := 100
	d := NewDistribution()
	items := b.N / concurrency
	if items < 1 {
		items = 1
	}
	count := 0
	doneCh := make(chan bool)
	for i := 0; i < b.N; i += items {
		go func(start int) {
			for j := start; j < start+items && j < b.N; j++ {
				d.Update(float64(int64(j) & 0xff))
			}
			doneCh <- true
		}(i)
		count++
	}
	for i := 0; i < count; i++ {
		_ = <-doneCh
	}
}
