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
	"github.com/samuel/stathatgo"
)

const (
	REPORT_INTERVAL = 60e9 // nanoseconds
)

var (
	f_laddr    = flag.String("l", "0.0.0.0:5252", "the address to listen on")
	f_perc     = flag.String("p", "0.90,0.99,0.999", "comma separated list of percentiles to record")
	f_username = flag.String("u", "", "librato metrics username")
	f_token    = flag.String("t", "", "librato metrics token")
	f_stathat  = flag.String("s", "", "StatHat email")
)

var (
	mu          sync.Mutex
	percentiles = []float64{}
	met         *librato.Metrics
)

func main() {
	parseFlags()
	metrics.AddSnapshotReceiver("60sec", reporter)
	go metrics.RunMetricsHeartbeat("60sec", 0,REPORT_INTERVAL * time.Nanosecond)
	packetLoop(listen())
}

func parseFlags() {
	flag.Parse()
	if *f_stathat == "" && (*f_username == "" || *f_token == "") {
		log.Fatal("Either StatHat email or Librato username & token required")
	}
	for _, s := range strings.Split(*f_perc, ",") {
		p, err := strconv.ParseFloat(s, 64)
		if err != nil {
			log.Fatal("Couldn't parse percentile flag: " + err.Error())
		}
		if p < 0 || p > 1 {
			log.Fatalf("Invalid percentile: %f", p)
		}
		percentiles = append(percentiles, p)
	}
	if *f_username != "" && *f_token != "" {
		met = &librato.Metrics{*f_username, *f_token}
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

			if mtype == 'c' {
				metrics.UpdateCounterf(name, value)
			} else if mtype == 't' {
				metrics.UpdateHistogram(name, value)
			}
		}
	}
}

func reporter(name string, snap metrics.Snapshot) {
	if len(snap.Counterfs) > 0 || len(snap.Histograms) > 0 {
		if met != nil {
			if err := sendMetricsLibrato(met, snap.Counterfs, snap.Histograms); err != nil {
				log.Printf(err.Error())
			}
		}

		if *f_stathat != "" {
			if err := sendMetricsStatHat(snap.Counterfs, snap.Histograms); err != nil {
				log.Printf(err.Error())
			}
		}
	}
}


func sendMetricsLibrato(met *librato.Metrics, counters map[string]float64, histograms map[string]*metrics.Histogram) error {
	metrics := librato.MetricsFormat{}
	for name, value := range counters {
		metrics.Counters = append(metrics.Counters, librato.Metric{Name: name, Value: value})
	}
	for name, hist := range histograms {
		metrics.Gauges = append(metrics.Gauges, librato.Metric{Name: name, Value: hist.GetMean()})
		for i, p := range hist.GetPercentiles(percentiles) {
			metrics.Gauges = append(metrics.Gauges,
				librato.Metric{Name: fmt.Sprintf("%s:%.2f", name, percentiles[i]*100), Value: p})
		}
	}

	return met.SendMetrics(&metrics)
}

func sendMetricsStatHat(counters map[string]float64, histograms map[string]*metrics.Histogram) error {
	for name, value := range counters {
		if err := stathat.PostEZCount(name, *f_stathat, int(value)); err != nil {
			return err
		}
	}
	for name, hist := range histograms {
		if err := stathat.PostEZValue(name, *f_stathat, hist.GetMean()); err != nil {
			return err
		}
		for i, p := range hist.GetPercentiles(percentiles) {
			if err := stathat.PostEZValue(fmt.Sprintf("%s:%.2f", name, percentiles[i]*100), *f_stathat, p); err != nil {
				return err
			}
		}
	}
	return nil
}
