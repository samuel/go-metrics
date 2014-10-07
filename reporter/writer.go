// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package reporter

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

type writerReporter struct {
	w io.Writer
}

func NewWriterReporter(registry metrics.Registry, interval time.Duration, latched bool, w io.Writer) *PeriodicReporter {
	return NewPeriodicReporter(registry, interval, false, latched, &writerReporter{w})
}

func (r *writerReporter) Report(snapshot *metrics.RegistrySnapshot) {
	fmt.Fprintf(r.w, "%+v\n", time.Now())
	for _, v := range snapshot.Values {
		if _, err := fmt.Fprintf(r.w, "%s: %f\n", v.Name, v.Value); err != nil {
			log.Printf("metricswriter: failed to post %s: %s", v.Name, err.Error())
		}
	}
	for _, v := range snapshot.Distributions {
		if _, err := fmt.Fprintf(r.w, "%s: %+v\n", v.Name, v.Value); err != nil {
			log.Printf("metricswriter: failed to post %s: %s", v.Name, err.Error())
		}
	}
}
