package metrics

// import "encoding/json"
import (
	"bytes"
	"fmt"
	"strconv"
)

var (
	defaultPercentiles     = []float64{0.5, 0.75, 0.9, 0.99, 0.999, 0.9999}
	defaultPrecentileNames = []string{"median", "p75", "p90", "p99", "p999", "p9999"}
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
	return histogramToJson(e.Histogram, e.Percentiles, e.PercentileNames)
}

// Return a JSON encoded version of the Histgram output
func histogramToJson(h Histogram, percentiles []float64, percentileNames []string) string {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "{\"count\":%d,\"sum\":%d,\"min\":%d,\"max\":%d,\"mean\":%s",
		h.Count(), h.Sum(), h.Min(), h.Max(), strconv.FormatFloat(h.Mean(), 'g', -1, 64))
	perc := h.Percentiles(percentiles)
	for i, p := range perc {
		fmt.Fprintf(b, ",\"%s\":%d", percentileNames[i], p)
	}
	fmt.Fprintf(b, "}")
	return b.String()
}
