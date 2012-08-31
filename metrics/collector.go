package metrics

import (
	"sync"
)

type GaugeFunc func() float64
type SampleBuilder func() Sample

type Collector struct {
	countersLock   sync.RWMutex
	counters       map[string]*Counter
	gaugesLock     sync.Mutex
	gauges         map[string]GaugeFunc
	histogramsLock sync.RWMutex
	histograms     map[string]Histogram
	metersLock     sync.RWMutex
	meters         map[string]*Meter
}

var (
	DefaultCollector       = NewCollector()
	DefaultPercentiles     = []float64{0.50, 0.75, 0.90, 0.95, 0.99, 0.999, 0.9999}
	DefaultPercentileNames = []string{"p50", "p75", "p90", "p95", "p99", "p999", "p9999"}
)

func NewCollector() *Collector {
	c := Collector{
		counters: make(map[string]*Counter),
		meters:   make(map[string]*Meter),
		gauges:   make(map[string]GaugeFunc),
	}
	return &c
}

func (c *Collector) Gauge(name string, f GaugeFunc) {
	c.gaugesLock.Lock()
	c.gauges[name] = f
	c.gaugesLock.Unlock()
}

func (c *Collector) Meter(name string) *Meter {
	c.metersLock.RLock()
	meter := c.meters[name]
	c.metersLock.RUnlock()
	if meter == nil {
		c.metersLock.Lock()
		// Need to check again to make sure no other go routine got here first
		meter = c.meters[name]
		if meter == nil {
			meter = NewMeter()
			c.meters[name] = meter
		}
		c.metersLock.Unlock()
	}
	return meter
}

func (c *Collector) Counter(name string) *Counter {
	c.countersLock.RLock()
	counter := c.counters[name]
	c.countersLock.RUnlock()
	if counter == nil {
		c.countersLock.Lock()
		// Need to check again to make sure no other go routine got here first
		counter = c.counters[name]
		if counter == nil {
			counter = NewCounter()
			c.counters[name] = counter
		}
		c.countersLock.Unlock()
	}
	return counter
}

func (c *Collector) Histogram(name string, sampleBuilder SampleBuilder) Histogram {
	c.histogramsLock.RLock()
	histogram := c.histograms[name]
	c.histogramsLock.RUnlock()
	if histogram == nil {
		c.histogramsLock.Lock()
		// Need to check again to make sure no other go routine got here first
		histogram = c.histograms[name]
		if histogram == nil {
			histogram = NewSampledHistogram(sampleBuilder())
			c.histograms[name] = histogram
		}
		c.histogramsLock.Unlock()
	}
	return histogram
}

func (c *Collector) BiasedHistogram(name string) Histogram {
	return c.Histogram(name, func() Sample { return NewExponentiallyDecayingSample(1028, 0.015) })
}

func (c *Collector) UnbiasedHistogram(name string) Histogram {
	return c.Histogram(name, func() Sample { return NewUniformSample(1028) })
}

func (c *Collector) Snapshot() *Snapshot {
	s := Snapshot{
		IntValues:   make(map[string]int64),
		FloatValues: make(map[string]float64),
	}

	c.countersLock.RLock()
	for name, counter := range c.counters {
		s.IntValues[name+".count"] = int64(counter.Count())
	}
	c.countersLock.RUnlock()

	c.metersLock.RLock()
	for name, meter := range c.meters {
		s.FloatValues[name+".rate-1m"] = meter.OneMinuteRate()
		s.FloatValues[name+".rate-5m"] = meter.FiveMinuteRate()
		s.FloatValues[name+".rate-15m"] = meter.FifteenMinuteRate()
		s.FloatValues[name+".rate-mean"] = meter.MeanRate()
	}
	c.metersLock.RUnlock()

	c.histogramsLock.RLock()
	for name, hist := range c.histograms {
		perc := hist.Percentiles(DefaultPercentiles)
		for i, p := range perc {
			s.IntValues[name+"."+DefaultPercentileNames[i]] = p
		}
		s.FloatValues[name+".mean"] = hist.Mean()
		s.IntValues[name+".min"] = hist.Min()
		s.IntValues[name+".max"] = hist.Max()
		s.IntValues[name+".count"] = int64(hist.Count())
	}
	c.histogramsLock.RUnlock()

	return &s
}
