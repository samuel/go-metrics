package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/samuel/go-librato"
	"github.com/samuel/go-metrics/metrics"
	"github.com/stathat/stathatgo"
)

const (
	REPORT_INTERVAL = 60e9 // nanoseconds
)

var (
	f_laddr    = flag.String("l", "0.0.0.0:5252", "the address to listen on")
	f_perc     = flag.String("p", "0.90,0.99,0.999", "comma separated list of percentiles to record")
	f_graphite = flag.String("g", "", "host:port for Graphite's Carbon")
	f_username = flag.String("u", "", "librato metrics username")
	f_token    = flag.String("t", "", "librato metrics token")
	f_stathat  = flag.String("s", "", "StatHat email")
)

var (
	mu          sync.Mutex
	counters    = make(map[string]float64)
	histograms  = make(map[string]*metrics.Histogram)
	percentiles = []float64{}
)

func main() {
	parseFlags()
	go reporter()
	packetLoop(listen())
}

func parseFlags() {
	flag.Parse()
	if *f_stathat == "" && (*f_username == "" || *f_token == "") && *f_graphite == "" {
		log.Fatal("Either StatHat email, Librato username & token, or Graphite/Carbon required")
	}
	for _, s := range strings.Split(*f_perc, ",") {
		p, err := strconv.ParseFloat(s, 64)
		switch {
		case err != nil:
			log.Fatal("Couldn't parse percentile flag: " + err.Error())
		case p < 0 || p > 1:
			log.Fatalf("Invalid percentile: %f", p)
		}
		percentiles = append(percentiles, p)
	}
}

func listen() *net.UDPConn {
	addr, err := net.ResolveUDPAddr("udp", *f_laddr)
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
		if err != nil {
			log.Println(err.Error())
		}
		if n > 9 {
			mtype := buf[0]
			var value float64
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

func updateCounter(name string, value float64) {
	mu.Lock()
	defer mu.Unlock()
	counters[name] += value
}

func updateHistogram(name string, value float64) {
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
	var met *librato.Metrics = nil
	if *f_username != "" && *f_token != "" {
		met = &librato.Metrics{*f_username, *f_token}
	}
	tc := time.Tick(REPORT_INTERVAL)
	for {
		ts := <-tc
		counters, histograms := swapMetrics()

		if len(counters) > 0 || len(histograms) > 0 {
			if *f_graphite != "" {
				if err := sendMetricsGraphite(ts, counters, histograms); err != nil {
					log.Printf(err.Error())
				}
			}

			if met != nil {
				if err := sendMetricsLibrato(met, ts, counters, histograms); err != nil {
					log.Printf(err.Error())
				}
			}

			if *f_stathat != "" {
				if err := sendMetricsStatHat(ts, counters, histograms); err != nil {
					log.Printf(err.Error())
				}
			}
		}
	}
}

func swapMetrics() (oldcounters map[string]float64, oldhistograms map[string]*metrics.Histogram) {
	mu.Lock()
	defer mu.Unlock()

	oldcounters = counters
	oldhistograms = histograms

	counters = make(map[string]float64)
	histograms = make(map[string]*metrics.Histogram)

	return
}

func sendMetricsGraphite(ts time.Time, counters map[string]float64, histograms map[string]*metrics.Histogram) error {
	conn, err := net.Dial("tcp", *f_graphite)
	if err != nil {
		return err
	}
	defer conn.Close()
	for name, value := range counters {
		if _, err := fmt.Fprintf(conn, "%s %f %d\n", name, value, ts.Unix()); err != nil {
			return err
		}
	}
	for name, hist := range histograms {
		for i, p := range hist.Percentiles(percentiles) {
			if _, err := fmt.Fprintf(conn, "%s:%.2f %f %d\n", name, percentiles[i]*100, p, ts.Unix()); err != nil {
				return err
			}
		}
	}

	return nil
}

func sendMetricsLibrato(met *librato.Metrics, ts time.Time, counters map[string]float64, histograms map[string]*metrics.Histogram) error {
	metrics := librato.MetricsFormat{}
	for name, value := range counters {
		metrics.Counters = append(metrics.Counters, librato.Metric{Name: name, Value: value})
	}
	for name, hist := range histograms {
		metrics.Gauges = append(metrics.Gauges, librato.Metric{Name: name, Value: hist.Mean()})
		for i, p := range hist.Percentiles(percentiles) {
			metrics.Gauges = append(metrics.Gauges,
				librato.Metric{Name: fmt.Sprintf("%s:%.2f", name, percentiles[i]*100), Value: p})
		}
	}

	return met.SendMetrics(&metrics)
}

func sendMetricsStatHat(ts time.Time, counters map[string]float64, histograms map[string]*metrics.Histogram) error {
	for name, value := range counters {
		if err := stathat.PostEZCount(name, *f_stathat, int(value)); err != nil {
			return err
		}
	}
	for name, hist := range histograms {
		if err := stathat.PostEZValue(name, *f_stathat, hist.Mean()); err != nil {
			return err
		}
		for i, p := range hist.Percentiles(percentiles) {
			if err := stathat.PostEZValue(fmt.Sprintf("%s:%.2f", name, percentiles[i]*100), *f_stathat, p); err != nil {
				return err
			}
		}
	}
	return nil
}
