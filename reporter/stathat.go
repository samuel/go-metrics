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
	source string
	email  string
}

func NewStatHatReporter(registry metrics.Registry, interval time.Duration, latched bool, email, source string) *PeriodicReporter {
	sr := &statHatReporter{
		source: source,
		email:  email,
	}
	return NewPeriodicReporter(registry, interval, false, latched, sr)
}

func (r *statHatReporter) Report(snapshot *metrics.RegistrySnapshot) {
	for _, v := range snapshot.Values {
		name := strings.Replace(v.Name, "/", ".", -1)
		if err := stathat.PostEZValue(name, r.email, v.Value); err != nil {
			log.Printf("stathat: failed to post metric %s: %s", name, err.Error())
		}
	}
	for _, v := range snapshot.Distributions {
		name := strings.Replace(v.Name, "/", ".", -1)
		if err := stathat.PostEZValue(name, r.email, v.Value.Mean()); err != nil {
			log.Printf("stathat: failed to post metric %s: %s", name, err.Error())
		}
	}
}
