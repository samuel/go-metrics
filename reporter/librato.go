// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package reporter

import (
	"log"
	"strings"
	"time"

	"github.com/samuel/go-librato/librato"
	"github.com/samuel/go-metrics/metrics"
)

type libratoReporter struct {
	source          string
	lib             *librato.Metrics
	percentiles     []float64
	percentileNames []string
	counterCache    *counterDeltaCache
}

func NewLibratoReporter(registry metrics.Registry, interval time.Duration, username, token, source string, percentiles map[string]float64) *PeriodicReporter {
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

	lr := &libratoReporter{
		source:          source,
		lib:             &librato.Metrics{Username: username, Token: token},
		percentiles:     per,
		percentileNames: perNames,
		counterCache:    &counterDeltaCache{},
	}
	return NewPeriodicReporter(registry, interval, true, lr)
}

func (r *libratoReporter) Report(registry metrics.Registry) {
	mets := &librato.MetricsFormat{Source: r.source}
	count := 0

	registry.Do(func(name string, metric interface{}) error {
		count++
		name = strings.Replace(name, "/", ".", -1)
		switch m := metric.(type) {
		case metrics.CounterValue:
			mets.Counters = append(mets.Counters,
				librato.Metric{
					Name:  name,
					Value: float64(m),
				})
		case metrics.GaugeValue:
			mets.Gauges = append(mets.Gauges,
				librato.Metric{
					Name:  name,
					Value: float64(m),
				})
		case metrics.IntegerGauge:
			mets.Gauges = append(mets.Gauges,
				librato.Metric{
					Name:  name,
					Value: float64(m.Value()),
				})
		case metrics.Counter:
			mets.Counters = append(mets.Counters,
				librato.Metric{
					Name:  name,
					Value: float64(m.Count()),
				})
		case *metrics.EWMA:
			mets.Gauges = append(mets.Gauges,
				librato.Metric{
					Name:  name,
					Value: m.Rate(),
				})
		case *metrics.EWMAGauge:
			mets.Gauges = append(mets.Gauges,
				librato.Metric{
					Name:  name,
					Value: m.Mean(),
				})
		case *metrics.Meter:
			mets.Gauges = append(mets.Gauges,
				librato.Metric{
					Name:  name + ".1m",
					Value: m.OneMinuteRate(),
				},
				librato.Metric{
					Name:  name + ".5m",
					Value: m.FiveMinuteRate(),
				},
				librato.Metric{
					Name:  name + ".15m",
					Value: m.FifteenMinuteRate(),
				})
		case metrics.Histogram:
			count := m.Count()
			if count > 0 {
				deltaCount := r.counterCache.delta(name+".count", int64(count))
				if deltaCount > 0 {
					deltaSum := r.counterCache.delta(name+".sum", m.Sum())
					mets.Gauges = append(mets.Gauges,
						librato.Gauge{
							Name:  name,
							Count: uint64(deltaCount),
							Sum:   float64(deltaSum),
						})
				}
				percentiles := m.Percentiles(r.percentiles)
				for i, perc := range percentiles {
					mets.Gauges = append(mets.Gauges,
						librato.Metric{
							Name:  name + "." + r.percentileNames[i],
							Value: float64(perc),
						})
				}
			}
		default:
			log.Printf("Unrecognized metric type for %s: %+v", name, m)
		}
		return nil
	})

	if count > 0 {
		if err := r.lib.SendMetrics(mets); err != nil {
			log.Printf("ERR librato.SendMetrics: %+v", err)
		}
	}
}
