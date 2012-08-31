package metrics

type Histogram interface {
	Clear()
	Update(value int64)
	Count() uint64
	Sum() int64
	Min() int64
	Max() int64
	Mean() float64
	Percentiles([]float64) []int64
}
