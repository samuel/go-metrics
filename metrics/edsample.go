package metrics

import (
	"container/heap"
	"fmt"
	"math"
	"math/rand"
	"time"
)

const (
	RESCALE_THRESHOLD = 360e9 // nanoseconds
)

// Reservoir

type priorityValue struct {
	priority float64
	value    float64
}

type reservoir struct {
	samples []priorityValue
}

func (self *reservoir) String() string {
	return fmt.Sprintf("%s", self.Values())
}

func (self *reservoir) Clear() {
	self.samples = self.samples[0:0]
}

func (self *reservoir) Get(i int) priorityValue {
	return self.samples[i]
}

func (self *reservoir) Values() (values []float64) {
	values = make([]float64, len(self.samples))
	for i, sample := range self.samples {
		values[i] = sample.value
	}
	return
}

func (self *reservoir) ScalePriority(scale float64) {
	for i, sample := range self.samples {
		self.samples[i] = priorityValue{sample.priority * scale, sample.value}
	}
}

func (self *reservoir) Len() int {
	return len(self.samples)
}

func (self *reservoir) Less(i, j int) bool {
	return self.samples[i].priority < self.samples[j].priority
}

func (self *reservoir) Swap(i, j int) {
	self.samples[i], self.samples[j] = self.samples[j], self.samples[i]
}

func (self *reservoir) Push(x interface{}) {
	self.samples = append(self.samples, x.(priorityValue))
}

func (self *reservoir) Pop() interface{} {
	v := self.samples[len(self.samples)-1]
	self.samples = self.samples[:len(self.samples)-1]
	return v
}

type exponentiallyDecayingSample struct {
	// the number of samples to keep in the sampling reservoir
	reservoirSize int
	// the exponential decay factor; the higher this is, the more
	// biased the sample will be towards newer values
	alpha         float64
	values        *reservoir
	count         int
	startTime     time.Time
	nextScaleTime time.Time
}

// An exponentially-decaying random sample of values. Uses Cormode et
// al's forward-decaying priority reservoir sampling method to produce a
// statistically representative sample, exponentially biased towards newer
// entries.
//
// http://www.research.att.com/people/Cormode_Graham/library/publications/CormodeShkapenyukSrivastavaXu09.pdf
// Cormode et al. Forward Decay: A Practical Time Decay Model for Streaming
// Systems. ICDE '09: Proceedings of the 2009 IEEE International Conference on
// Data Engineering (2009)
func NewExponentiallyDecayingSample(reservoirSize int, alpha float64) Sample {
	eds := exponentiallyDecayingSample{
		reservoirSize: reservoirSize,
		alpha:         alpha,
		values:        &reservoir{}}
	eds.Clear()
	return &eds
}

func (self *exponentiallyDecayingSample) Clear() {
	self.values.Clear()
	heap.Init(self.values)
	self.count = 0
	self.startTime = time.Now()
	self.nextScaleTime = self.startTime.Add(RESCALE_THRESHOLD)
}

func (self *exponentiallyDecayingSample) Len() int {
	vl := self.values.Len()
	if self.count < vl {
		return self.count
	}
	return vl
}

func (self *exponentiallyDecayingSample) Values() []float64 {
	return self.values.Values()
}

func (self *exponentiallyDecayingSample) Update(value float64) {
	timestamp := time.Now()
	priority := self.weight(timestamp.Sub(self.startTime)) / rand.Float64()
	self.count += 1
	if self.count <= self.reservoirSize {
		heap.Push(self.values, priorityValue{priority, value})
	} else {
		if first := self.values.Get(0); first.priority > priority {
			// heap.Replace(self.values, priorityValue{priority, value})
			heap.Pop(self.values)
			heap.Push(self.values, priorityValue{priority, value})
		}
	}

	if timestamp.After(self.nextScaleTime) {
		self.rescale(timestamp)
	}
}

func (self *exponentiallyDecayingSample) weight(delta time.Duration) float64 {
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
func (self *exponentiallyDecayingSample) rescale(now time.Time) {
	self.nextScaleTime = now.Add(RESCALE_THRESHOLD)
	oldStartTime := self.startTime
	self.startTime = now
	scale := math.Exp(-self.alpha * float64(self.startTime.Sub(oldStartTime)))
	self.values.ScalePriority(scale)
}
