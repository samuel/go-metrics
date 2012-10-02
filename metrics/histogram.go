package metrics

// import "encoding/json"
import (
	"bytes"
	"fmt"
	"strconv"
)

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
	h := e.Histogram
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "{\"count\":%d,\"sum\":%d,\"min\":%d,\"max\":%d,\"mean\":%s",
		h.Count(), h.Sum(), h.Min(), h.Max(), strconv.FormatFloat(h.Mean(), 'g', -1, 64))
	perc := h.Percentiles(e.Percentiles)
	for i, p := range perc {
		fmt.Fprintf(b, ",\"%s\":%d", e.PercentileNames[i], p)
	}
	fmt.Fprintf(b, "}")
	return b.String()
}
