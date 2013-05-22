// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Meter struct {
	count          uint64
	m1Rate         *EWMA
	m5Rate         *EWMA
	m15Rate        *EWMA
	startTime      time.Time
	ticker         *time.Ticker
	tickerStopChan chan bool
}

func NewMeter() *Meter {
	interval := time.Second * 5
	m := Meter{
		m1Rate:         NewEWMA(interval, M1Alpha),
		m5Rate:         NewEWMA(interval, M5Alpha),
		m15Rate:        NewEWMA(interval, M15Alpha),
		ticker:         time.NewTicker(interval),
		tickerStopChan: make(chan bool),
	}
	go m.tickWatcher()
	return &m
}

func (m *Meter) String() string {
	return fmt.Sprintf("{\"1\": %s, \"5\": %s, \"15\": %s}",
		m.m1Rate.String(), m.m5Rate.String(), m.m15Rate.String())
}

func (m *Meter) tickWatcher() {
watcher:
	for {
		select {
		case _ = <-m.tickerStopChan:
			break watcher
		case _ = <-m.ticker.C:
			m.tick()
		}
	}
	m.ticker = nil
	m.tickerStopChan = nil
}

func (m *Meter) tick() {
	m.m1Rate.Tick()
	m.m5Rate.Tick()
	m.m15Rate.Tick()
}

// Stop the ticker
func (m *Meter) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
		close(m.tickerStopChan)
	}
}

func (m *Meter) Update(delta uint64) {
	atomic.AddUint64(&m.count, delta)
	m.m1Rate.Update(delta)
	m.m5Rate.Update(delta)
	m.m15Rate.Update(delta)
}

func (m *Meter) Count() uint64 {
	return atomic.LoadUint64(&m.count)
}

func (m *Meter) MeanRate() float64 {
	tdelta := time.Now().Sub(m.startTime)
	count := m.Count()
	return float64(count) / tdelta.Seconds()
}

func (m *Meter) OneMinuteRate() float64 {
	return m.m1Rate.Rate()
}

func (m *Meter) FiveMinuteRate() float64 {
	return m.m5Rate.Rate()
}

func (m *Meter) FifteenMinuteRate() float64 {
	return m.m15Rate.Rate()
}
