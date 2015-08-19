package reporter

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

type influxDBReporter struct {
	writeURL string
	tags     string
}

// NewInfluxDBReporter returns a new period reporter that sends metrics to InfluxDB.
// BaseURL should be of the form http://localhost:8086
func NewInfluxDBReporter(registry metrics.Registry, interval time.Duration, latched bool, baseURL, dbName string, tags map[string]string) *PeriodicReporter {
	if len(baseURL) == 0 {
		baseURL = "http://localhost:8086"
	} else if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	var tagStr string
	if len(tags) != 0 {
		for k, v := range tags {
			tagStr += fmt.Sprintf(",%s=%s", k, v)
		}
	}
	lr := &influxDBReporter{
		writeURL: fmt.Sprintf("%s/write?db=%s", baseURL, dbName),
		tags:     tagStr,
	}
	return NewPeriodicReporter(registry, interval, true, latched, lr)
}

func (r *influxDBReporter) Report(snapshot *metrics.RegistrySnapshot) {
	var measurements []string
	for _, v := range snapshot.Values {
		name := strings.Replace(v.Name, "/", "_", -1)
		measurements = append(measurements, name+r.tags+" value="+strconv.FormatFloat(v.Value, 'f', -1, 64))
	}
	for _, v := range snapshot.Distributions {
		name := strings.Replace(v.Name, "/", "_", -1)
		if v.Value.Count != 0 {
			measurements = append(measurements, name+r.tags+fmt.Sprintf(" count=%di,sum=%f,min=%f,max=%f,variance=%f", v.Value.Count, v.Value.Sum, v.Value.Min, v.Value.Max, v.Value.Variance))
		}
	}
	body := strings.Join(measurements, "\n")
	res, err := http.Post(r.writeURL, "application/binary", strings.NewReader(body))
	if err != nil {
		log.Printf("ERR influxdb.PostMetrics: %+v", err)
	} else {
		res.Body.Close()
	}
}
