package metrics

import (
	"math"
	"time"
	"sync/atomic"
)

const (
	M1_ALPHA  = 0.07995558537067670723530454779393039643764495849609 // 1 - math.Exp(-5 / 60.0)
	M5_ALPHA  = 0.01652854617838250828043555884505622088909149169922 // 1 - math.Exp(-5 / 60.0 / 5)
	M15_ALPHA = 0.00554015199510327072118798241717740893363952636719 // 1 - math.Exp(-5 / 60.0 / 15)
)

// An exponentially-weighted moving average.
//
// http://www.teamquest.com/pdfs/whitepaper/ldavg1.pdf - UNIX Load Average Part 1: How It Works
// http://www.teamquest.com/pdfs/whitepaper/ldavg2.pdf - UNIX Load Average Part 2: Not Your Average Average
type EWMA struct {
	interval       time.Duration // tick interval in seconds
	alpha          float64       // the smoothing constant
	uncounted      uint64
	rate           uint64 // really a float64 but using uint64 for atomicity
	ticker         *time.Ticker
	tickerStopChan chan bool
}

func NewEWMA(interval time.Duration, alpha float64) *EWMA {
	ewma := &EWMA{
		interval: interval,
		alpha:    alpha,
	}
	return ewma
}

// Increment the uncounted value - thread safe
func (ewma *EWMA) Update(value uint64) {
	atomic.AddUint64(&ewma.uncounted, value)
}

// Return the rate - thread safe
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
	if rate == 0.0 {
		rate = instantRate
	} else {
		rate += ewma.alpha * (instantRate - rate)
	}
	atomic.StoreUint64(&ewma.rate, math.Float64bits(rate))
}
