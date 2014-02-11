// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math"
	"strconv"
	"sync/atomic"
	"time"
)

// FloatGaugeFunc is used for stats reporting to identify the value as a floating point gauge
type FloatGaugeFunc func() float64

type EWMAGauge struct {
	mean           uint64        // really a float64 but using uint64 for atomicity
	alpha          float64       // the smoothing constant
	interval       time.Duration // tick interval in seconds
	initialized    bool
	ticker         *time.Ticker
	tickerStopChan chan bool
	fun            FloatGaugeFunc
}

func NewEWMAGauge(interval time.Duration, alpha float64, fun FloatGaugeFunc) *EWMAGauge {
	ewma := &EWMAGauge{
		interval:    interval,
		alpha:       alpha,
		initialized: false,
		fun:         fun,
	}
	return ewma
}

func (e *EWMAGauge) String() string {
	rate := e.Mean()
	return strconv.FormatFloat(rate, 'g', -1, 64)
}

func (e *EWMAGauge) Mean() float64 {
	return math.Float64frombits(atomic.LoadUint64(&e.mean))
}

// Start the ticker
func (e *EWMAGauge) Start() {
	if e.ticker == nil {
		e.ticker = time.NewTicker(e.interval)
		e.tickerStopChan = make(chan bool)
		go e.tickWatcher()
	}
}

// Stop the ticker
func (e *EWMAGauge) Stop() {
	if e.ticker != nil {
		e.ticker.Stop()
		close(e.tickerStopChan)
	}
}

func (e *EWMAGauge) tickWatcher() {
	defer func() {
		e.ticker.Stop()
		e.ticker = nil
		e.tickerStopChan = nil
	}()
	for {
		select {
		case _ = <-e.tickerStopChan:
			return
		case _ = <-e.ticker.C:
			e.Tick()
		}
	}
}

// Tick the moving average - NOT thread safe
func (e *EWMAGauge) Tick() {
	value := e.fun()
	mean := e.Mean()
	if e.initialized {
		mean += e.alpha * (value - mean)
	} else {
		mean = value
		e.initialized = true
	}
	atomic.StoreUint64(&e.mean, math.Float64bits(mean))
}
