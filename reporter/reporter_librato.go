package reporter

import (
	"log"
	"strings"
	"time"

	"github.com/samuel/go-librato"
	"github.com/samuel/go-metrics/metrics"
)

type LibratoReporter struct {
	source          string
	registry        *metrics.Registry
	interval        time.Duration
	ticker          *time.Ticker
	closeChan       chan bool
	lib             *librato.Metrics
	percentiles     []float64
	percentileNames []string
}

func NewLibratoReporter(registry *metrics.Registry, interval time.Duration, username, token, source string, percentiles map[string]float64) *LibratoReporter {
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

	return &LibratoReporter{
		source:          source,
		lib:             &librato.Metrics{username, token},
		registry:        registry,
		interval:        interval,
		percentiles:     per,
		percentileNames: perNames,
	}
}

func nsToNextInterval(t time.Time, i time.Duration) time.Duration {
	return time.Duration(int64(i) - (int64(t.UnixNano()) % int64(i)))
}

func (r *LibratoReporter) Start() {
	if r.ticker == nil {
		go func() {
			// Wait until the beginning of the next even interval. Librato-Metrics
			// is sensitive to multiple entries falling on the exact same second.

			// Calculate nanoseconds to start of next interval and sleep.
			time.Sleep(nsToNextInterval(time.Now(), r.interval))

			r.closeChan = make(chan bool)
			r.ticker = time.NewTicker(r.interval)
			go r.recorder(r.ticker.C)
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

func (r *LibratoReporter) recorder(ch <-chan time.Time) {
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
					mets.Gauges = append(mets.Gauges,
						librato.Gauge{
							Name:  name,
							Count: count,
							Sum:   float64(m.Sum()),
							Min:   float64(m.Min()),
							Max:   float64(m.Max()),
						})
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
}
