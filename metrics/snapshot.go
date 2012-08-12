package metrics

type Snapshot struct {
	IntValues   map[string]int64
	FloatValues map[string]float64
}
