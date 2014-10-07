// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package reporter

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

type graphiteReporter struct {
	addr   string
	source string
}

func NewGraphiteReporter(registry metrics.Registry, interval time.Duration, latched bool, addr, source string) *PeriodicReporter {
	gr := &graphiteReporter{
		addr:   addr,
		source: source,
	}
	return NewPeriodicReporter(registry, interval, false, latched, gr)
}

func (r *graphiteReporter) sourcedName(name string) string {
	if r.source != "" {
		return name + "." + r.source
	}
	return name
}

func (r *graphiteReporter) Report(snapshot *metrics.RegistrySnapshot) {
	conn, err := net.Dial("tcp", r.addr)
	if err != nil {
		log.Printf("Failed to connect to graphite/carbon: %+v", err)
		return
	}
	defer conn.Close()

	ts := time.Now().UTC().Unix()

	for _, v := range snapshot.Values {
		name := strings.Replace(v.Name, "/", ".", -1)
		if _, err := fmt.Fprintf(conn, "%s %f %d\n", r.sourcedName(name), v.Value, ts); err != nil {
			log.Printf("graphite: failed to post metric %s: %s", name, err.Error())
		}
	}
	for _, v := range snapshot.Distributions {
		name := strings.Replace(v.Name, "/", ".", -1)
		if _, err := fmt.Fprintf(conn, "%s %f %d\n", r.sourcedName(name), v.Value.Mean(), ts); err != nil {
			log.Printf("graphite: failed to post metric %s: %s", name, err.Error())
		}
	}
}
