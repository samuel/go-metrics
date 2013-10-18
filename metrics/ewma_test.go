// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

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
	{M1Alpha, 0, 0.60000000}, {M1Alpha, 1, 0.22072766},
	{M1Alpha, 2, 0.08120117}, {M1Alpha, 3, 0.02987224},
	{M1Alpha, 4, 0.01098938}, {M1Alpha, 5, 0.00404277},
	{M1Alpha, 6, 0.00148725}, {M1Alpha, 7, 0.00054713},
	{M1Alpha, 8, 0.00020128}, {M1Alpha, 9, 0.00007405},
	{M1Alpha, 10, 0.00002724}, {M1Alpha, 11, 0.00001002},
	{M1Alpha, 12, 0.00000369}, {M1Alpha, 13, 0.00000136},
	{M1Alpha, 14, 0.00000050}, {M1Alpha, 15, 0.00000018},

	{M5Alpha, 0, 0.60000000}, {M5Alpha, 1, 0.49123845},
	{M5Alpha, 2, 0.40219203}, {M5Alpha, 3, 0.32928698},
	{M5Alpha, 4, 0.26959738}, {M5Alpha, 5, 0.22072766},
	{M5Alpha, 6, 0.18071653}, {M5Alpha, 7, 0.14795818},
	{M5Alpha, 8, 0.12113791}, {M5Alpha, 9, 0.09917933},
	{M5Alpha, 10, 0.08120117}, {M5Alpha, 11, 0.06648190},
	{M5Alpha, 12, 0.05443077}, {M5Alpha, 13, 0.04456415},
	{M5Alpha, 14, 0.03648604}, {M5Alpha, 15, 0.02987224},

	{M15Alpha, 0, 0.60000000}, {M15Alpha, 1, 0.56130419},
	{M15Alpha, 2, 0.52510399}, {M15Alpha, 3, 0.49123845},
	{M15Alpha, 4, 0.45955700}, {M15Alpha, 5, 0.42991879},
	{M15Alpha, 6, 0.40219203}, {M15Alpha, 7, 0.37625345},
	{M15Alpha, 8, 0.35198773}, {M15Alpha, 9, 0.32928698},
	{M15Alpha, 10, 0.30805027}, {M15Alpha, 11, 0.28818318},
	{M15Alpha, 12, 0.26959738}, {M15Alpha, 13, 0.25221023},
	{M15Alpha, 14, 0.23594443}, {M15Alpha, 15, 0.22072766},
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
	e := NewEWMA(time.Millisecond*50, M1Alpha)
	e.Start()
	if e.ticker == nil {
		t.Errorf("EWMA.ticker should not be nil")
	}
	time.Sleep(time.Millisecond * 100)
	e.Stop()
}

func BenchmarkEWMAUpdate(b *testing.B) {
	e := NewEWMA(time.Second*5, M1Alpha)
	for i := 0; i < b.N; i++ {
		e.Update(1)
	}
}

func BenchmarkEWMARate(b *testing.B) {
	e := NewEWMA(time.Second*5, M1Alpha)
	for i := 0; i < b.N; i++ {
		e.Rate()
	}
}

func BenchmarkEWMAConcurrentUpdate(b *testing.B) {
	concurrency := 100
	e := NewEWMA(time.Second*5, M1Alpha)
	items := b.N / concurrency
	if items < 1 {
		items = 1
	}
	count := 0
	doneCh := make(chan bool)
	for i := 0; i < b.N; i += items {
		go func(start int) {
			for j := start; j < start+items && j < b.N; j++ {
				e.Update(1)
			}
			doneCh <- true
		}(i)
		count++
	}
	for i := 0; i < count; i++ {
		_ = <-doneCh
	}
}
