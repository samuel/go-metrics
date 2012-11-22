package metrics

import "runtime"

var SystemMetrics systemMetrics

type systemMetrics runtime.MemStats

func (s *systemMetrics) Metrics() map[string]interface{} {
	m := (*runtime.MemStats)(s)
	runtime.ReadMemStats(m)
	return map[string]interface{}{
		"Mallocs":         CounterValue(m.Mallocs),
		"Frees":           CounterValue(m.Frees),
		"heap/HeapAlloc":  GaugeValue(m.HeapAlloc),
		"gc/NumGC":        CounterValue(m.NumGC),
		"gc/PauseTotalNs": CounterValue(m.PauseTotalNs),
	}
}
