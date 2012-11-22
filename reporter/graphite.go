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
	addr             string
	percentiles      []float64
	percentileNames  []string
	previousCounters map[string]int64 // TODO: These should expire if counters aren't seen again
}

func NewGraphiteReporter(registry *metrics.Registry, interval time.Duration, addr string, percentiles map[string]float64) *PeriodicReporter {
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

	gr := &graphiteReporter{
		addr:             addr,
		percentiles:      per,
		percentileNames:  perNames,
		previousCounters: make(map[string]int64),
	}
	return NewPeriodicReporter(registry, interval, false, gr)
}

func (r *graphiteReporter) Report(registry *metrics.Registry) {
	conn, err := net.Dial("tcp", r.addr)
	if err != nil {
		log.Printf("Failed to connect to graphite/carbon: %+v", err)
		return
	}
	defer conn.Close()

	ts := time.Now().UTC().Unix()

	err = registry.Do(func(name string, metric interface{}) error {
		name = strings.Replace(name, "/", ".", -1)
		switch m := metric.(type) {
		case metrics.CounterValue:
			count := int64(m)
			prev := r.previousCounters[name]
			r.previousCounters[name] = count
			if _, err := fmt.Fprintf(conn, "%s %d %d\n", name, count-prev, ts); err != nil {
				return err
			}
		case metrics.GaugeValue:
			if _, err := fmt.Fprintf(conn, "%s %f %d\n", name, m, ts); err != nil {
				return err
			}
		case metrics.Counter:
			count := m.Count()
			prev := r.previousCounters[name]
			r.previousCounters[name] = count
			if _, err := fmt.Fprintf(conn, "%s %d %d\n", name, count-prev, ts); err != nil {
				return err
			}
		case *metrics.EWMA:
			if _, err := fmt.Fprintf(conn, "%s %f %d\n", name, m.Rate(), ts); err != nil {
				return err
			}
		case *metrics.Meter:
			if _, err := fmt.Fprintf(conn, "%s.1m %f %d\n", name, m.OneMinuteRate(), ts); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(conn, "%s.5m %f %d\n", name, m.FiveMinuteRate(), ts); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(conn, "%s.15m %f %d\n", name, m.FifteenMinuteRate(), ts); err != nil {
				return err
			}
		case metrics.Histogram:
			count := m.Count()
			if count > 0 {
				if _, err := fmt.Fprintf(conn, "%s.mean %f %d\n", name, m.Mean(), ts); err != nil {
					return err
				}
				percentiles := m.Percentiles(r.percentiles)
				for i, perc := range percentiles {
					if _, err := fmt.Fprintf(conn, "%s.%s %d %d\n", name, r.percentileNames[i], perc, ts); err != nil {
						return err
					}
				}
			}
		default:
			log.Printf("Unrecognized metric type for %s: %+v", name, m)
		}
		return nil
	})
	if err != nil {
		log.Printf("ERR graphite: %+v", err)
	}
}
