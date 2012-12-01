// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import "testing"

func BenchmarkDistributionConcurrentUpdate(b *testing.B) {
	concurrency := 1000
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
				d.Update(int64(j) & 0xff)
			}
			doneCh <- true
		}(i)
		count++
	}
	for i := 0; i < count; i++ {
		_ = <-doneCh
	}
}
