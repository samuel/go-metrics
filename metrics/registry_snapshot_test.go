package metrics

import (
	"sort"
	"testing"
)

type namedValueSlice []NamedValue

func (s namedValueSlice) Len() int {
	return len(s)
}

func (s namedValueSlice) Less(a, b int) bool {
	return s[a].Name < s[b].Name
}

func (s namedValueSlice) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

func TestRegistrySnapshot(t *testing.T) {
	reg := NewRegistry()
	snap := NewRegistrySnapshot()

	snap.Snapshot(reg)

	counter := NewCounter()
	counter.Inc(2)
	reg.Add("counter", counter)
	gauge := NewIntegerGauge()
	gauge.Inc(3)
	reg.Add("gauge", gauge)
	hist := NewUnbiasedHistogram()
	hist.Update(2.0)
	reg.Add("hist", hist)

	snap.Snapshot(reg)
	sort.Sort(namedValueSlice(snap.Values))
	t.Logf("%+v", snap)

	if len(snap.Values) != 7 {
		t.Fatalf("Expected 7 values. Got %d", len(snap.Values))
	}
	if e := (NamedValue{Name: "counter", Value: 2}); snap.Values[0] != e {
		t.Errorf("Expected %+v. Got %+v", e, snap.Values[0])
	}
	if e := (NamedValue{Name: "gauge", Value: 3}); snap.Values[1] != e {
		t.Errorf("Expected %+v. Got %+v", e, snap.Values[1])
	}

	counter.Inc(1)
	gauge.Set(4)

	snap.Snapshot(reg)
	sort.Sort(namedValueSlice(snap.Values))
	t.Logf("%+v", snap)

	if e := (NamedValue{Name: "counter", Value: 1}); snap.Values[0] != e {
		t.Errorf("Expected %+v. Got %+v", e, snap.Values[0])
	}
	if e := (NamedValue{Name: "gauge", Value: 4}); snap.Values[1] != e {
		t.Errorf("Expected %+v. Got %+v", e, snap.Values[1])
	}
}
