// An exponentially-decaying random sample of {@code long}s. Uses Cormode et
// al's forward-decaying priority reservoir sampling method to produce a
// statistically representative sample, exponentially biased towards newer
// entries.
// 
// http://www.research.att.com/people/Cormode_Graham/library/publications/CormodeShkapenyukSrivastavaXu09.pdf
// Cormode et al. Forward Decay: A Practical Time Decay Model for Streaming
// Systems. ICDE '09: Proceedings of the 2009 IEEE International Conference on
// Data Engineering (2009)

package metrics

import (
    "fmt"
    "math"
    "rand"
    "time"
    "container/heap"
)

const (
    RESCALE_THRESHOLD = 360e9 // nanoseconds
)

// Reservoir

type PriorityValue struct {
    priority float64
    value    float64
}

type Reservoir struct {
    samples []PriorityValue
}

func (self *Reservoir) String() string {
    return fmt.Sprintf("%s", self.GetValues())
}

func (self *Reservoir) Clear() {
    self.samples = self.samples[0:0]
}

func (self *Reservoir) Get(i int) PriorityValue {
    return self.samples[i]
}

func (self *Reservoir) GetValues() []float64 {
    values := make([]float64, len(self.samples))
    for i := 0; i < len(self.samples); i++ {
        values[i] = self.samples[i].value
    }
    return values
}

func (self *Reservoir) ScalePriority(scale float64) {
    for i, sample := range self.samples {
        self.samples[i] = PriorityValue{sample.priority * scale, sample.value}
    }
}

func (self *Reservoir) Len() int {
    return len(self.samples)
}

func (self *Reservoir) Less(i, j int) bool {
    return self.samples[i].priority < self.samples[j].priority
}

func (self *Reservoir) Swap(i, j int) {
    a := self.samples[i]
    self.samples[i] = self.samples[j]
    self.samples[j] = a
}

func (self *Reservoir) Push(x interface{}) {
    self.samples = append(self.samples, x.(PriorityValue))
}

func (self *Reservoir) Pop() interface{} {
    v := self.samples[len(self.samples)-1]
    self.samples = self.samples[:len(self.samples)-1]
    return v
}

// ExponentiallyDecayingSample

type ExponentiallyDecayingSample struct {
    // the number of samples to keep in the sampling reservoir
    reservoir_size  int
    // the exponential decay factor; the higher this is, the more
    // biased the sample will be towards newer values
    alpha           float64
    values          *Reservoir
    count           int
    start_time      int64
    next_scale_time int64
}

func NewExponentiallyDecayingSample(reservoir_size int, alpha float64) *ExponentiallyDecayingSample {
    eds := ExponentiallyDecayingSample{
        reservoir_size: reservoir_size,
        alpha:          alpha,
        values:         &Reservoir{}}
    eds.Clear()
    return &eds
}

func (self *ExponentiallyDecayingSample) Clear() {
    self.values.Clear()
    heap.Init(self.values)
    self.count = 0
    self.start_time = time.Nanoseconds()
    self.next_scale_time = self.start_time + RESCALE_THRESHOLD
}

func (self *ExponentiallyDecayingSample) Len() int {
    vl := self.values.Len()
    if self.count < vl {
        return self.count
    }
    return vl
}

func (self *ExponentiallyDecayingSample) GetValues() []float64 {
    return self.values.GetValues()
}

func (self *ExponentiallyDecayingSample) Update(value float64) {
    timestamp := time.Nanoseconds()
    priority := self.weight(timestamp-self.start_time) / rand.Float64()
    self.count += 1
    if self.count <= self.reservoir_size {
        heap.Push(self.values, PriorityValue{priority, value})
    } else {
        first := self.values.Get(0)
        if first.priority > priority {
            // heap.Replace(self.values, PriorityValue{priority, value})
            heap.Pop(self.values)
            heap.Push(self.values, PriorityValue{priority, value})
        }
    }

    if timestamp >= self.next_scale_time {
        self.rescale(timestamp)
    }
}

func (self *ExponentiallyDecayingSample) weight(delta int64) float64 {
    return math.Exp(self.alpha * float64(delta))
}

/*
A common feature of the above techniques—indeed, the key technique that
allows us to track the decayed weights efficiently—is that they maintain
counts and other quantities based on g(ti − L), and only scale by g(t − L)
at query time. But while g(ti −L)/g(t−L) is guaranteed to lie between zero
and one, the intermediate values of g(ti − L) could become very large. For
polynomial functions, these values should not grow too large, and should be
effectively represented in practice by floating point values without loss of
precision. For exponential functions, these values could grow quite large as
new values of (ti − L) become large, and potentially exceed the capacity of
common floating point types. However, since the values stored by the
algorithms are linear combinations of g values (scaled sums), they can be
rescaled relative to a new landmark. That is, by the analysis of exponential
decay in Section III-A, the choice of L does not affect the final result. We
can therefore multiply each value based on L by a factor of exp(−α(L′ − L)),
and obtain the correct value as if we had instead computed relative to a new
landmark L′ (and then use this new L′ at query time). This can be done with
a linear pass over whatever data structure is being used.
*/
func (self *ExponentiallyDecayingSample) rescale(now int64) {
    self.next_scale_time = now + RESCALE_THRESHOLD
    old_start_time := self.start_time
    self.start_time = now
    scale := math.Exp(-self.alpha * float64(self.start_time-old_start_time))
    self.values.ScalePriority(scale)
}
