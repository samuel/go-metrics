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

var (
	// M1Alpha represents 1 minute at a 5 second interval
	M1Alpha = 1 - math.Exp(-5.0/60/1)
	// M5Alpha represents 5 minutes at a 5 second interval
	M5Alpha = 1 - math.Exp(-5.0/60/5)
	// M15Alpha represents 15 minutes at a 5 second interval
	M15Alpha = 1 - math.Exp(-5.0/60/15)
)

// EWMA is an exponentially-weighted moving average.
//
// http://www.teamquest.com/pdfs/whitepaper/ldavg1.pdf - UNIX Load Average Part 1: How It Works
// http://www.teamquest.com/pdfs/whitepaper/ldavg2.pdf - UNIX Load Average Part 2: Not Your Average Average
type EWMA struct {
	interval       time.Duration // tick interval in seconds
	rate           uint64        // really a float64 but using uint64 for atomicity
	alpha          float64       // the smoothing constant
	uncounted      uint64
	initialized    bool
	ticker         *time.Ticker
	tickerStopChan chan bool
}

// NewEWMA returns a new exponentially-weighte moving average.
func NewEWMA(interval time.Duration, alpha float64) *EWMA {
	return &EWMA{
		interval:    interval,
		alpha:       alpha,
		initialized: false,
	}
}

func (e *EWMA) String() string {
	rate := e.Rate()
	return strconv.FormatFloat(rate, 'g', -1, 64)
}

func (e *EWMA) MarshalJSON() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *EWMA) MarshalText() ([]byte, error) {
	return e.MarshalJSON()
}

// Update increments the uncounted value
func (e *EWMA) Update(value uint64) {
	atomic.AddUint64(&e.uncounted, value)
}

// Rate retusnt the current rate
func (e *EWMA) Rate() float64 {
	return math.Float64frombits(atomic.LoadUint64(&e.rate))
}

// Start the ticker
func (e *EWMA) Start() {
	if e.ticker == nil {
		e.ticker = time.NewTicker(e.interval)
		e.tickerStopChan = make(chan bool)
		go e.tickWatcher()
	}
}

// Stop the ticker
func (e *EWMA) Stop() {
	if e.ticker != nil {
		e.ticker.Stop()
		close(e.tickerStopChan)
	}
}

func (e *EWMA) tickWatcher() {
watcher:
	for {
		select {
		case _ = <-e.tickerStopChan:
			break watcher
		case _ = <-e.ticker.C:
			e.Tick()
		}
	}
	e.ticker = nil
	e.tickerStopChan = nil
}

// Tick the moving average
func (e *EWMA) Tick() {
	// Assume Tick is never called concurrently
	count := atomic.SwapUint64(&e.uncounted, 0)
	instantRate := float64(count) / e.interval.Seconds()
	rate := e.Rate()
	if e.initialized {
		rate += e.alpha * (instantRate - rate)
	} else {
		rate = instantRate
		e.initialized = true
	}
	atomic.StoreUint64(&e.rate, math.Float64bits(rate))
}
