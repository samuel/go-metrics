package reporter

import (
	"os"
	"testing"
	"time"

	"github.com/samuel/go-metrics/metrics"
)

func TestCloudWatch(t *testing.T) {
	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Skip("Missing AWS_ACCESS_KEY or AWS_SECRET_KEY environment variable")
	}
	registry := metrics.NewRegistry()
	hist := metrics.NewUnbiasedHistogram()
	hist.Update(100)
	hist.Update(120)
	hist.Update(300)
	hist.Update(50)
	hist.Update(123)
	registry.Add("Test", hist)
	auth := func() (string, string, string) {
		return accessKey, secretKey, ""
	}
	reporter := newCloudWatchReporter(time.Minute, "us-east-1", auth, "Test", map[string]string{"Test": "go-metrics"}, map[string]float64{"p50": 0.5}, time.Second*10)
	reporter.Report(registry)
}
