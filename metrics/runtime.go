// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import "runtime"

var RuntimeMetrics = &runtimeMetrics{}

type runtimeMetrics struct {
	memStats runtime.MemStats
}

func (s *runtimeMetrics) Metrics() map[string]interface{} {
	runtime.ReadMemStats(&s.memStats)
	return map[string]interface{}{
		"Mallocs":          CounterValue(s.memStats.Mallocs),
		"Frees":            CounterValue(s.memStats.Frees),
		"heap/HeapAlloc":   GaugeValue(s.memStats.HeapAlloc),
		"heap/HeapObjects": GaugeValue(s.memStats.HeapObjects),
		"gc/NumGC":         CounterValue(s.memStats.NumGC),
		"gc/PauseTotalNs":  CounterValue(s.memStats.PauseTotalNs),
		"Goroutines":       GaugeValue(runtime.NumGoroutine()),
		"CgoCalls":         CounterValue(runtime.NumCgoCall()),
	}
}
