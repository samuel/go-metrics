package metrics

import (
    "math"
    "sort"
)

type Sample interface {
    Clear()
    Len() int
    GetValues() []float64
    Update(value float64)
}

type Histogram struct {
    sample     Sample
    min        float64
    max        float64
    sum        float64
    count      int
    variance_m float64
    variance_s float64
}

func NewHistogram(sample Sample) *Histogram {
    return &Histogram{
        sample:     sample,
        min:        0,
        max:        0,
        sum:        0,
        count:      0,
        variance_m: 0,
        variance_s: 0}
}

/*
  Uses an exponentially decaying sample of 1028 elements, which offers
  a 99.9% confidence level with a 5% margin of error assuming a normal
  distribution, and an alpha factor of 0.015, which heavily biases
  the sample to the past 5 minutes of measurements.
*/
func NewBiasedHistogram(reservoir_size int) *Histogram {
    return NewHistogram(NewExponentiallyDecayingSample(1028, 0.015))
}

/*
  Uses a uniform sample of 1028 elements, which offers a 99.9%
  confidence level with a 5% margin of error assuming a normal
  distribution.
*/
func NewUnbiasedHistogram(reservoir_size int) *Histogram {
    return NewHistogram(NewUniformSample(1028))
}

func (self *Histogram) Clear() {
    self.sample.Clear()
    self.min = 0
    self.max = 0
    self.sum = 0
    self.count = 0
    self.variance_m = 0
    self.variance_s = 0
}

func (self *Histogram) Update(value float64) {
    self.count += 1
    self.sum += value
    self.sample.Update(value)
    if self.count == 1 {
        self.min = value
        self.max = value
        self.variance_m = value
    } else {
        if value < self.min {
            self.min = value
        }
        if value > self.max {
            self.max = value
        }
        old_m := self.variance_m
        self.variance_m = old_m + ((value - old_m) / float64(self.count))
        self.variance_s += (value - old_m) * (value - self.variance_m)
    }
}

func (self *Histogram) GetCount() int {
    return self.count
}

func (self *Histogram) GetSum() float64 {
    return self.sum
}

func (self *Histogram) GetMin() float64 {
    if self.count == 0 {
        return math.NaN()
    }
    return self.min
}

func (self *Histogram) GetMax() float64 {
    if self.count == 0 {
        return math.NaN()
    }
    return self.max
}

func (self *Histogram) GetMean() float64 {
    if self.count > 0 {
        return self.sum / float64(self.count)
    }
    return 0
}

func (self *Histogram) GetStdDev() float64 {
    if self.count > 0 {
        return math.Sqrt(self.variance_s / float64(self.count-1))
    }
    return 0
}

func (self *Histogram) GetVariance() float64 {
    if self.count <= 1 {
        return 0
    }
    return self.variance_s / float64(self.count-1)
}

func (self *Histogram) GetPercentiles(percentiles []float64) []float64 {
    scores := make([]float64, len(percentiles))
    if self.count == 0 {
        return scores
    }

    values := sort.Float64Slice(self.sample.GetValues())
    sort.Sort(values)
    for i, p := range percentiles {
        pos := p * float64(len(values)+1)
        ipos := int(pos)
        if ipos < 1 {
            scores[i] = values[0]
        } else if ipos >= len(values) {
            scores[i] = values[len(values)-1]
        } else {
            lower := values[ipos-1]
            upper := values[ipos]
            scores[i] = lower + (pos-math.Floor(pos))*(upper-lower)
        }
    }

    return scores
}

func (self *Histogram) GetValues() []float64 {
    return self.sample.GetValues()
}
