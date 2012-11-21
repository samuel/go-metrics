package reporter

import (
	"log"
	"strings"
	"time"

	"github.com/samuel/go-librato"
	"github.com/samuel/go-metrics/metrics"
)

type LibratoReporter struct {
	source    string
	registry  *metrics.Registry
	interval  time.Duration
	ticker    *time.Ticker
	closeChan chan bool
	lib       *librato.Metrics
}

func NewLibratoReporter(registry *metrics.Registry, interval time.Duration, username, token, source string) *LibratoReporter {
	return &LibratoReporter{
		source:   source,
		lib:      &librato.Metrics{username, token},
		registry: registry,
		interval: interval,
	}
}

func (r *LibratoReporter) Start() {
	if r.ticker == nil {
		r.ticker = time.NewTicker(r.interval)
		r.closeChan = make(chan bool)
		ch := r.ticker.C
		go func() {
			for {
				select {
				case <-ch:
				case <-r.closeChan:
					return
				}

				mets := &librato.MetricsFormat{Source: r.source}
				count := 0

				r.registry.Do(func(name string, metric interface{}) error {
					count++
					name = strings.Replace(name, "/", ".", -1)
					switch m := metric.(type) {
					case metrics.Counter:
						mets.Counters = append(mets.Counters,
							librato.Metric{
								// Source: r.source,
								Name:  name,
								Value: float64(m.Count()),
							})
					case metrics.Histogram:
						count := m.Count()
						if count > 0 {
							mets.Gauges = append(mets.Gauges,
								librato.Gauge{
									// Source: r.source,
									Name:  name,
									Count: count,
									Sum:   float64(m.Sum()),
									Min:   float64(m.Min()),
									Max:   float64(m.Max()),
								})
						} else {
							mets.Gauges = append(mets.Gauges,
								librato.Metric{
									// Source: r.source,
									Name:  name,
									Value: 0,
								})
						}
					}
					return nil
				})

				if count > 0 {
					if err := r.lib.SendMetrics(mets); err != nil {
						log.Printf("ERR librato.SendMetrics: %+v", err)
					}
				}
			}
		}()
	}
}

func (r *LibratoReporter) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
		close(r.closeChan)
		r.ticker = nil
	}
}
