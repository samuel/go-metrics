// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import "testing"

func TestIntegerGauge(t *testing.T) {
	c := NewIntegerGauge()
	if c.Value() != 0 {
		t.Fatalf("IntegerGauge initial value should be 0 not %d", c.Value())
	}
	c.Inc(2)
	if c.Value() != 2 {
		t.Fatalf("IntegerGauge inc should have made value 2 not %d", c.Value())
	}
	c.Dec(1)
	if c.Value() != 1 {
		t.Fatalf("IntegerGauge dec should have made value 1 not %d", c.Value())
	}
}

func BenchmarkIntegerGaugeInc(b *testing.B) {
	c := NewIntegerGauge()
	for i := 0; i < b.N; i++ {
		c.Inc(1)
	}
}

func BenchmarkIntegerGaugeValue(b *testing.B) {
	c := NewIntegerGauge()
	for i := 0; i < b.N; i++ {
		c.Value()
	}
}

func BenchmarkGaugeSet(b *testing.B) {
	c := NewIntegerGauge()
	for i := 0; i < b.N; i++ {
		c.Set(1)
	}
}
