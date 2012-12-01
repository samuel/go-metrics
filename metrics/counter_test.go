// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import "testing"

func TestCounter(t *testing.T) {
	c := NewCounter()
	if c.Count() != 0 {
		t.Fatalf("Counter initial value should be 0 not %d", c.Count())
	}
	c.Inc(2)
	if c.Count() != 2 {
		t.Fatalf("Counter inc should have made value 2 not %d", c.Count())
	}
	c.Dec(1)
	if c.Count() != 1 {
		t.Fatalf("Counter dec should have made value 1 not %d", c.Count())
	}
}

func BenchmarkCounterInc(b *testing.B) {
	c := NewCounter()
	for i := 0; i < b.N; i++ {
		c.Inc(1)
	}
}

func BenchmarkCounterCount(b *testing.B) {
	c := NewCounter()
	for i := 0; i < b.N; i++ {
		c.Count()
	}
}

func BenchmarkCounterSet(b *testing.B) {
	c := NewCounter()
	for i := 0; i < b.N; i++ {
		c.Set(1)
	}
}
