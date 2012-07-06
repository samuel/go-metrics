package metrics

type Snapshot struct {
	IntValues   map[string]uint64
	FloatValues map[string]float64
}
