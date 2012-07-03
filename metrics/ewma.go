package metrics

import (
	"time"
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
	uncounted      float64
	rate           float64
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

func (ewma *EWMA) Update(value float64) {
	ewma.uncounted += value
}

func (ewma *EWMA) Rate() float64 {
	return ewma.rate
}

func (ewma *EWMA) Start() {
	if ewma.ticker == nil {
		ewma.ticker = time.NewTicker(ewma.interval)
		ewma.tickerStopChan = make(chan bool)
		go ewma.tickWatcher()
	}
}

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

func (ewma *EWMA) Tick() {
	count := ewma.uncounted
	ewma.uncounted = 0
	instantRate := count / ewma.interval.Seconds()
	if ewma.rate == 0.0 {
		ewma.rate = instantRate
	} else {
		ewma.rate += ewma.alpha * (instantRate - ewma.rate)
	}
}
