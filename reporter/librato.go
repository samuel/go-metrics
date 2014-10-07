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
	source string
	client *librato.Client
}

func NewLibratoReporter(registry metrics.Registry, interval time.Duration, latched bool, username, token, source string) *PeriodicReporter {
	lr := &libratoReporter{
		source: source,
		client: &librato.Client{Username: username, Token: token},
	}
	return NewPeriodicReporter(registry, interval, true, latched, lr)
}

func (r *libratoReporter) Report(snapshot *metrics.RegistrySnapshot) {
	mets := &librato.Metrics{Source: r.source}

	for _, v := range snapshot.Values {
		name := strings.Replace(v.Name, "/", ".", -1)
		mets.Gauges = append(mets.Gauges, librato.Metric{Name: name, Value: v.Value})
	}
	for _, v := range snapshot.Distributions {
		name := strings.Replace(v.Name, "/", ".", -1)
		if v.Value.Count == 0 {
			mets.Gauges = append(mets.Gauges, librato.Metric{Name: name, Value: 0.0})
		} else {
			mets.Gauges = append(mets.Gauges,
				librato.Gauge{
					Name:       name,
					Count:      v.Value.Count,
					Sum:        v.Value.Sum,
					Min:        v.Value.Min,
					Max:        v.Value.Max,
					SumSquares: v.Value.Variance,
				})
		}
	}

	if len(mets.Gauges) > 0 {
		if err := r.client.PostMetrics(mets); err != nil {
			log.Printf("ERR librato.PostMetrics: %+v", err)
		}
	}
}
