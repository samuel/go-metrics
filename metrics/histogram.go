// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package metrics

import (
	"bytes"
	"fmt"
	"strconv"
)

var (
	DefaultPercentiles     = []float64{0.5, 0.75, 0.9, 0.99, 0.999}
	DefaultPercentileNames = []string{"p50", "p75", "p90", "p99", "p999"}
)

type Histogram interface {
	Clear()
	Update(int64)
	Distribution() DistributionValue
	Percentiles([]float64) []int64
	String() string
}

type HistogramExport struct {
	Histogram       Histogram
	Percentiles     []float64
	PercentileNames []string
}

type histogramValues struct {
	count       uint64
	sum         int64
	min         int64
	max         int64
	mean        float64
	percentiles map[string]int64
}

// Return a JSON encoded version of the Histgram output
func (e *HistogramExport) String() string {
	return histogramToJSON(e.Histogram, e.Percentiles, e.PercentileNames)
}

func (e *HistogramExport) MarshalJSON() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *HistogramExport) MarshalText() ([]byte, error) {
	return e.MarshalJSON()
}

// Return a JSON encoded version of the Histgram output
func histogramToJSON(h Histogram, percentiles []float64, percentileNames []string) string {
	v := h.Distribution()
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "{\"count\":%d,\"sum\":%f,\"min\":%f,\"max\":%f,\"mean\":%s",
		v.Count, v.Sum, v.Min, v.Max, strconv.FormatFloat(v.Mean(), 'g', -1, 64))
	perc := h.Percentiles(percentiles)
	for i, p := range perc {
		fmt.Fprintf(b, ",\"%s\":%d", percentileNames[i], p)
	}
	fmt.Fprintf(b, "}")
	return b.String()
}
