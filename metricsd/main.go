package main

import (
    "bytes"
    "encoding/binary"
    "flag"
    "fmt"
    "log"
    "os"
    "net"
    "strconv"
    "strings"
    "sync"
    "time"

    "librato"
    "metrics"
)

const (
    REPORT_INTERVAL = 60e9 // nanoseconds
)

var (
    f_laddr    = flag.String("l", "0.0.0.0:5252", "the address to listen on")
    f_perc     = flag.String("p", "0.90,0.95,0.99,0.999", "comma separated list of percentiles to record")
    f_username = flag.String("u", "", "librato metrics username")
    f_token    = flag.String("t", "", "librato metrics token")
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
    if *f_username == "" {
        log.Fatal("username is required (-u)")
    }
    if *f_token == "" {
        log.Fatal("token is required (-t)")
    }
    for _, s := range strings.Split(*f_perc, ",") {
        p, err := strconv.Atof64(s)
        if err != nil {
            log.Fatal("Couldn't parse percentile flag: " + err.String())
        }
        if p < 0 || p > 1 {
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
            log.Println(err.String())
        }
        if n > 9 {
            mtype := buf[0]
            var value float64
            binary.Read(bytes.NewBuffer(buf[1:9]), binary.BigEndian, &value)
            name := string(buf[9:n])

            if mtype == 'c' {
                updateCounter(name, value)
            } else if mtype == 't' {
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
    met := librato.Metrics{*f_username, *f_token}
    tc := time.Tick(REPORT_INTERVAL)
    for {
        ts := <-tc
        counters, histograms := swapMetrics()

        err := sendMetrics(met, ts, counters, histograms)
        if err != nil {
            log.Printf(err.String())
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

func sendMetrics(met librato.Metrics, ts int64, counters map[string]float64, histograms map[string]*metrics.Histogram) os.Error {
    if len(counters) == 0 && len(histograms) == 0 {
        return nil
    }

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
