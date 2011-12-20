package metrics

import "math"

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func almostEqual(a, b, diff float64) bool {
	return math.Abs(a-b) < diff
}
