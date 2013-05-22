package metrics

import (
	"testing"
)

func TestMeter(t *testing.T) {
	m := NewMeter()
	m.Update(1)
	m.Update(10)
	m.Count()
	m.MeanRate()
	m.Stop()
}
