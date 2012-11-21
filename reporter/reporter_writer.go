package reporter

import (
	"fmt"
	"io"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

type WriterReporter struct {
	registry  *metrics.Registry
	interval  time.Duration
	ticker    *time.Ticker
	closeChan chan bool
	w         io.Writer
}

func NewWriterReporter(registry *metrics.Registry, interval time.Duration, w io.Writer) *WriterReporter {
	return &WriterReporter{
		w:        w,
		registry: registry,
		interval: interval,
	}
}

func (r *WriterReporter) Start() {
	if r.ticker == nil {
		r.ticker = time.NewTicker(r.interval)
		r.closeChan = make(chan bool)
		ch := r.ticker.C
		go func() {
			for {
				var ts time.Time
				select {
				case ts = <-ch:
				case <-r.closeChan:
					return
				}
				fmt.Fprintf(r.w, "%+v\n", ts)
				r.registry.Do(func(name string, metric interface{}) error {
					_, err := fmt.Fprintf(r.w, "%s: %+v\n", name, metric)
					return err
				})
			}
		}()
	}
}

func (r *WriterReporter) Stop() {
	if r.ticker != nil {
		r.ticker.Stop()
		close(r.closeChan)
		r.ticker = nil
	}
}
