package metrics

import (
	"math"
)

// Int64Slice attaches the methods of sort.Interface to []float64, sorting in increasing order.
type Int64Slice []int64

func (s Int64Slice) Len() int {
	return len(s)
}

func (s Int64Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s Int64Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func almostEqual(a, b, diff float64) bool {
	return math.Abs(a-b) < diff
}
