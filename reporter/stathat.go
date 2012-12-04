// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package reporter

import (
	"log"
	"strings"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/stathat/stathatgo"
)

type statHatReporter struct {
	source          string
	email           string
	percentiles     []float64
	percentileNames []string
	counterCache    *counterDeltaCache
}

func NewStatHatReporter(registry metrics.Registry, interval time.Duration, email, source string, percentiles map[string]float64) *PeriodicReporter {
	per := metrics.DefaultPercentiles
	perNames := metrics.DefaultPercentileNames

	if percentiles != nil {
		per = make([]float64, 0)
		perNames = make([]string, 0)
		for name, p := range percentiles {
			per = append(per, p)
			perNames = append(perNames, name)
		}
	}

	sr := &statHatReporter{
		source:          source,
		email:           email,
		percentiles:     per,
		percentileNames: perNames,
		counterCache:    &counterDeltaCache{},
	}
	return NewPeriodicReporter(registry, interval, false, sr)
}

func (r *statHatReporter) Report(registry metrics.Registry) {
	registry.Do(func(name string, metric interface{}) error {
		name = strings.Replace(name, "/", ".", -1)
		switch m := metric.(type) {
		case metrics.CounterValue:
			if err := stathat.PostEZCount(name, r.email, int(r.counterCache.delta(name, int64(m)))); err != nil {
				log.Printf("ERR stathat.PostEZCount: %+v", err)
			}
		case metrics.GaugeValue:
			if err := stathat.PostEZValue(name, r.email, float64(m)); err != nil {
				log.Printf("ERR stathat.PostEZValue: %+v", err)
			}
		case metrics.Counter:
			if err := stathat.PostEZCount(name, r.email, int(r.counterCache.delta(name, m.Count()))); err != nil {
				log.Printf("ERR stathat.PostEZCount: %+v", err)
			}
		case *metrics.EWMA:
			if err := stathat.PostEZValue(name, r.email, m.Rate()); err != nil {
				log.Printf("ERR stathat.PostEZValue: %+v", err)
			}
		case *metrics.EWMAGauge:
			if err := stathat.PostEZValue(name, r.email, m.Mean()); err != nil {
				log.Printf("ERR stathat.PostEZValue: %+v", err)
			}
		case *metrics.Meter:
			if err := stathat.PostEZValue(name+".1m", r.email, m.OneMinuteRate()); err != nil {
				log.Printf("ERR stathat.PostEZValue: %+v", err)
			}
			if err := stathat.PostEZValue(name+".5m", r.email, m.FiveMinuteRate()); err != nil {
				log.Printf("ERR stathat.PostEZValue: %+v", err)
			}
			if err := stathat.PostEZValue(name+".15m", r.email, m.FifteenMinuteRate()); err != nil {
				log.Printf("ERR stathat.PostEZValue: %+v", err)
			}
		case metrics.Histogram:
			count := m.Count()
			if count > 0 {
				deltaCount := r.counterCache.delta(name+".count", int64(count))
				if deltaCount > 0 {
					deltaSum := r.counterCache.delta(name+".sum", m.Sum())
					if err := stathat.PostEZValue(name+".mean", r.email, float64(deltaSum)/float64(deltaCount)); err != nil {
						log.Printf("ERR stathat.PostEZValue: %+v", err)
					}
				}
				percentiles := m.Percentiles(r.percentiles)
				for i, perc := range percentiles {
					if err := stathat.PostEZValue(name+"."+r.percentileNames[i], r.email, float64(perc)); err != nil {
						log.Printf("ERR stathat.PostEZValue: %+v", err)
					}
				}
			}
		default:
			log.Printf("Unrecognized metric type for %s: %+v", name, m)
		}
		return nil
	})
}
