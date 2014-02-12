// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"expvar"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/samuel/go-librato/librato"
	"github.com/samuel/go-metrics/metrics"
	"github.com/stathat/stathatgo"
)

const (
	reportInterval = 60e9 // nanoseconds
)

var (
	flagHTTPAddr        = flag.String("h", "0.0.0.0:5251", "address of HTTP server")
	flagListenAddr      = flag.String("l", "0.0.0.0:5252", "the address to listen on")
	flagPercentiles     = flag.String("p", "0.90,0.99,0.999", "comma separated list of percentiles to record")
	flagGraphite        = flag.String("g", "", "host:port for Graphite's Carbon")
	flagLibratoUsername = flag.String("u", "", "librato metrics username")
	flagLibratoToken    = flag.String("t", "", "librato metrics token")
	flagStatHatEmail    = flag.String("s", "", "StatHat email")
)

var (
	mu              sync.Mutex
	counters        = make(map[string]int64)
	histograms      = make(map[string]metrics.Histogram)
	percentiles     = []float64{}
	percentileNames = []string{}

	statRequestCount    = metrics.NewCounter()
	statRequestRate     = metrics.NewMeter()
	statGraphiteLatency = metrics.NewBiasedHistogram()
	statLibratoLatency  = metrics.NewBiasedHistogram()
	statStatHatLatency  = metrics.NewBiasedHistogram()
)

func init() {
	m := expvar.NewMap("metricsd")
	m.Set("requests", statRequestCount)
	m.Set("requests_per_sec", statRequestRate)
	m.Set("graphite_latency_us", &metrics.HistogramExport{Histogram: statGraphiteLatency,
		Percentiles: []float64{0.5, 0.9, 0.99, 0.999}, PercentileNames: []string{"p50", "p90", "p99", "p999"}})
	m.Set("librato_latency_us", &metrics.HistogramExport{Histogram: statLibratoLatency,
		Percentiles: []float64{0.5, 0.9, 0.99, 0.999}, PercentileNames: []string{"p50", "p90", "p99", "p999"}})
	m.Set("stathat_latency_us", &metrics.HistogramExport{Histogram: statStatHatLatency,
		Percentiles: []float64{0.5, 0.9, 0.99, 0.999}, PercentileNames: []string{"p50", "p90", "p99", "p999"}})
}

func main() {
	parseFlags()

	if *flagHTTPAddr != "" {
		go func() {
			log.Fatal(http.ListenAndServe(*flagHTTPAddr, nil))
		}()
	}

	go reporter()
	packetLoop(listen())
}

func parseFlags() {
	flag.Parse()
	if *flagStatHatEmail == "" && (*flagLibratoUsername == "" || *flagLibratoToken == "") && *flagGraphite == "" {
		log.Fatal("Either StatHat email, Librato username & token, or Graphite/Carbon required")
	}
	for _, s := range strings.Split(*flagPercentiles, ",") {
		p, err := strconv.ParseFloat(s, 64)
		switch {
		case err != nil:
			log.Fatal("Couldn't parse percentile flag: " + err.Error())
		case p < 0 || p > 1:
			log.Fatalf("Invalid percentile: %f", p)
		}
		percentiles = append(percentiles, p)
		percentileNames = append(percentileNames, strings.Replace(s, "0.", "p", 1))
	}
}

func listen() *net.UDPConn {
	addr, err := net.ResolveUDPAddr("udp", *flagListenAddr)
	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	return l
}

func packetLoop(l net.PacketConn) {
	buf := make([]byte, 4096)
	for {
		n, _, err := l.ReadFrom(buf)
		statRequestCount.Inc(1)
		statRequestRate.Update(1)
		if err != nil {
			log.Println(err.Error())
		}
		if n > 9 {
			mtype := buf[0]
			var value int64
			binary.Read(bytes.NewBuffer(buf[1:9]), binary.BigEndian, &value)
			name := string(buf[9:n])

			switch mtype {
			case 'c':
				updateCounter(name, value)
			case 't':
				updateHistogram(name, value)
			}
		}
	}
}

func updateCounter(name string, value int64) {
	mu.Lock()
	defer mu.Unlock()
	counters[name] += value
}

func updateHistogram(name string, value int64) {
	mu.Lock()
	defer mu.Unlock()
	hist := histograms[name]
	if hist == nil {
		hist = metrics.NewUnbiasedHistogram()
		histograms[name] = hist
	}
	hist.Update(value)
}

func reporter() {
	var met *librato.Client
	if *flagLibratoUsername != "" && *flagLibratoToken != "" {
		met = &librato.Client{Username: *flagLibratoUsername, Token: *flagLibratoToken}
	}
	tc := time.Tick(reportInterval)
	for {
		ts := <-tc
		counters, histograms := swapMetrics()

		if len(counters) > 0 || len(histograms) > 0 {
			if *flagGraphite != "" {
				startTime := time.Now()
				if err := sendMetricsGraphite(ts, counters, histograms); err != nil {
					log.Printf(err.Error())
				}
				statGraphiteLatency.Update(time.Now().Sub(startTime).Nanoseconds() / 1e3)
			}

			if met != nil {
				startTime := time.Now()
				if err := sendMetricsLibrato(met, ts, counters, histograms); err != nil {
					log.Printf(err.Error())
				}
				statLibratoLatency.Update(time.Now().Sub(startTime).Nanoseconds() / 1e3)
			}

			if *flagStatHatEmail != "" {
				startTime := time.Now()
				if err := sendMetricsStatHat(ts, counters, histograms); err != nil {
					log.Printf(err.Error())
				}
				statStatHatLatency.Update(time.Now().Sub(startTime).Nanoseconds() / 1e3)
			}
		}
	}
}

func swapMetrics() (oldcounters map[string]int64, oldhistograms map[string]metrics.Histogram) {
	mu.Lock()
	defer mu.Unlock()

	oldcounters = counters
	oldhistograms = histograms

	counters = make(map[string]int64)
	histograms = make(map[string]metrics.Histogram)

	return
}

func sendMetricsGraphite(ts time.Time, counters map[string]int64, histograms map[string]metrics.Histogram) error {
	conn, err := net.Dial("tcp", *flagGraphite)
	if err != nil {
		return err
	}
	defer conn.Close()
	for name, value := range counters {
		if _, err := fmt.Fprintf(conn, "%s %d %d\n", name, value, ts.Unix()); err != nil {
			return err
		}
	}
	for name, hist := range histograms {
		for i, p := range hist.Percentiles(percentiles) {
			if _, err := fmt.Fprintf(conn, "%s.%s %d %d\n", name, percentileNames[i], p, ts.Unix()); err != nil {
				return err
			}
		}
	}

	return nil
}

func sendMetricsLibrato(met *librato.Client, ts time.Time, counters map[string]int64, histograms map[string]metrics.Histogram) error {
	var metrics librato.Metrics
	for name, value := range counters {
		metrics.Counters = append(metrics.Counters, librato.Metric{Name: name, Value: float64(value)})
	}
	for name, hist := range histograms {
		metrics.Gauges = append(metrics.Gauges, librato.Metric{Name: name, Value: hist.Mean()})
		for i, p := range hist.Percentiles(percentiles) {
			metrics.Gauges = append(metrics.Gauges,
				librato.Metric{Name: fmt.Sprintf("%s.%s", name, percentileNames[i]), Value: float64(p)})
		}
	}

	return met.SendMetrics(&metrics)
}

func sendMetricsStatHat(ts time.Time, counters map[string]int64, histograms map[string]metrics.Histogram) error {
	for name, value := range counters {
		if err := stathat.PostEZCount(name, *flagStatHatEmail, int(value)); err != nil {
			return err
		}
	}
	for name, hist := range histograms {
		if err := stathat.PostEZValue(name, *flagStatHatEmail, hist.Mean()); err != nil {
			return err
		}
		for i, p := range hist.Percentiles(percentiles) {
			if err := stathat.PostEZValue(fmt.Sprintf("%s.%s", name, percentileNames[i]), *flagStatHatEmail, float64(p)); err != nil {
				return err
			}
		}
	}
	return nil
}
