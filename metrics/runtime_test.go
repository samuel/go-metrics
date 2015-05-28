package metrics

import "testing"

func TestRuntimeMetrics(t *testing.T) {
	if m := RuntimeMetrics.Metrics(); len(m) == 0 {
		t.Fatal("RuntimeMetrics returned no values")
	}
}
