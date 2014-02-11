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
	ewma := &EWMA{
		interval:    interval,
		alpha:       alpha,
		initialized: false,
	}
	return ewma
}

func (ewma *EWMA) String() string {
	rate := ewma.Rate()
	return strconv.FormatFloat(rate, 'g', -1, 64)
}

// Update increments the uncounted value
func (ewma *EWMA) Update(value uint64) {
	atomic.AddUint64(&ewma.uncounted, value)
}

// Rate retusnt the current rate
func (ewma *EWMA) Rate() float64 {
	return math.Float64frombits(atomic.LoadUint64(&ewma.rate))
}

// Start the ticker
func (ewma *EWMA) Start() {
	if ewma.ticker == nil {
		ewma.ticker = time.NewTicker(ewma.interval)
		ewma.tickerStopChan = make(chan bool)
		go ewma.tickWatcher()
	}
}

// Stop the ticker
func (ewma *EWMA) Stop() {
	if ewma.ticker != nil {
		ewma.ticker.Stop()
		close(ewma.tickerStopChan)
	}
}

func (ewma *EWMA) tickWatcher() {
watcher:
	for {
		select {
		case _ = <-ewma.tickerStopChan:
			break watcher
		case _ = <-ewma.ticker.C:
			ewma.Tick()
		}
	}
	ewma.ticker = nil
	ewma.tickerStopChan = nil
}

// Tick the moving average - NOT thread safe
func (ewma *EWMA) Tick() {
	// Assume Tick is never called concurrently
	count := atomic.LoadUint64(&ewma.uncounted)
	// Subtract the old count since there is no atomic get-and-set
	atomic.AddUint64(&ewma.uncounted, -count)
	instantRate := float64(count) / ewma.interval.Seconds()
	rate := ewma.Rate()
	if ewma.initialized {
		rate += ewma.alpha * (instantRate - rate)
	} else {
		rate = instantRate
		ewma.initialized = true
	}
	atomic.StoreUint64(&ewma.rate, math.Float64bits(rate))
}
