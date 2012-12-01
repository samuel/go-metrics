// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"math"
)

// Int64Slice attaches the methods of sort.Interface to []float64, sorting in increasing order.
type int64Slice []int64

func (s int64Slice) Len() int {
	return len(s)
}

func (s int64Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s int64Slice) Swap(i, j int) {
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
