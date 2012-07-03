package metrics

import (
	"testing"
	"time"
)

type testStruct struct {
	alpha    float64
	minutes  int
	expected float64
}

var testData = []testStruct{
	{M1_ALPHA, 0, 0.60000000}, {M1_ALPHA, 1, 0.22072766},
	{M1_ALPHA, 2, 0.08120117}, {M1_ALPHA, 3, 0.02987224},
	{M1_ALPHA, 4, 0.01098938}, {M1_ALPHA, 5, 0.00404277},
	{M1_ALPHA, 6, 0.00148725}, {M1_ALPHA, 7, 0.00054713},
	{M1_ALPHA, 8, 0.00020128}, {M1_ALPHA, 9, 0.00007405},
	{M1_ALPHA, 10, 0.00002724}, {M1_ALPHA, 11, 0.00001002},
	{M1_ALPHA, 12, 0.00000369}, {M1_ALPHA, 13, 0.00000136},
	{M1_ALPHA, 14, 0.00000050}, {M1_ALPHA, 15, 0.00000018},

	{M5_ALPHA, 0, 0.60000000}, {M5_ALPHA, 1, 0.49123845},
	{M5_ALPHA, 2, 0.40219203}, {M5_ALPHA, 3, 0.32928698},
	{M5_ALPHA, 4, 0.26959738}, {M5_ALPHA, 5, 0.22072766},
	{M5_ALPHA, 6, 0.18071653}, {M5_ALPHA, 7, 0.14795818},
	{M5_ALPHA, 8, 0.12113791}, {M5_ALPHA, 9, 0.09917933},
	{M5_ALPHA, 10, 0.08120117}, {M5_ALPHA, 11, 0.06648190},
	{M5_ALPHA, 12, 0.05443077}, {M5_ALPHA, 13, 0.04456415},
	{M5_ALPHA, 14, 0.03648604}, {M5_ALPHA, 15, 0.02987224},

	{M15_ALPHA, 0, 0.60000000}, {M15_ALPHA, 1, 0.56130419},
	{M15_ALPHA, 2, 0.52510399}, {M15_ALPHA, 3, 0.49123845},
	{M15_ALPHA, 4, 0.45955700}, {M15_ALPHA, 5, 0.42991879},
	{M15_ALPHA, 6, 0.40219203}, {M15_ALPHA, 7, 0.37625345},
	{M15_ALPHA, 8, 0.35198773}, {M15_ALPHA, 9, 0.32928698},
	{M15_ALPHA, 10, 0.30805027}, {M15_ALPHA, 11, 0.28818318},
	{M15_ALPHA, 12, 0.26959738}, {M15_ALPHA, 13, 0.25221023},
	{M15_ALPHA, 14, 0.23594443}, {M15_ALPHA, 15, 0.22072766},
}

func TestEWMA(t *testing.T) {
	for _, data := range testData {
		e := NewEWMA(time.Second*5, data.alpha)
		e.Update(3)
		e.Tick()
		for i := 0; i < data.minutes*60/5; i++ {
			e.Tick()
		}
		if !almostEqual(e.Rate(), data.expected, 0.00000001) {
			t.Errorf("EWMA alpha=%.8f minutes=%d expected=%.8f != %.8f", data.alpha, data.minutes, data.expected, e.Rate())
		}
	}
}

func TestEWMATicker(t *testing.T) {
	e := NewEWMA(time.Millisecond*50, M1_ALPHA)
	e.Start()
	if e.ticker == nil {
		t.Errorf("EWMA.ticker should not be nil")
	}
	time.Sleep(time.Millisecond * 100)
	e.Stop()
	time.Sleep(time.Millisecond * 50)
	if e.ticker != nil {
		t.Errorf("EWMA.ticker should be nil")
	}
}
