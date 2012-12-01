// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package reporter

import (
	"time"

	"github.com/samuel/go-metrics/metrics"
)

type PeriodicReporter struct {
	registry      metrics.Registry
	interval      time.Duration
	alignInterval bool
	ticker        *time.Ticker
	closeChan     chan bool
	reporter      Reporter
}

type Reporter interface {
	Report(registry metrics.Registry)
}

func NewPeriodicReporter(registry metrics.Registry, interval time.Duration, alignInterval bool, reporter Reporter) *PeriodicReporter {
	return &PeriodicReporter{
		registry:      registry,
		interval:      interval,
		alignInterval: alignInterval,
		reporter:      reporter,
	}
}

// Calculate nanoseconds to start of next interval
func nsToNextInterval(t time.Time, i time.Duration) time.Duration {
	return time.Duration(int64(i) - (int64(t.UnixNano()) % int64(i)))
}

func (r *PeriodicReporter) Start() {
	if r.ticker == nil {
		go func() {
			if r.alignInterval {
				// Wait until the beginning of the next even interval. This gives
				// a better chance that different sources for the same metric
				// will fall on the same timestamp.
				time.Sleep(nsToNextInterval(time.Now(), r.interval))
			}

			r.closeChan = make(chan bool)
			r.ticker = time.NewTicker(r.interval)
			go r.loop(r.ticker.C)
		}()
	}
}

func (r *PeriodicReporter) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
		close(r.closeChan)
		r.ticker = nil
	}
}

func (r *PeriodicReporter) loop(ch <-chan time.Time) {
	for {
		select {
		case <-ch:
		case <-r.closeChan:
			return
		}
		r.reporter.Report(r.registry)
	}
}
