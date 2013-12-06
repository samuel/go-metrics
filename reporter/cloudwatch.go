package reporter

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bmizerany/aws4"
	"github.com/samuel/go-metrics/metrics"
)

type cloudWatchReporter struct {
	namespace       string
	percentiles     []float64
	percentileNames []string
	counterCache    *counterDeltaCache
	client          *aws4.Client
	dimensions      map[string]string
	endpoint        string
	securityToken   string
}

type cloudWatchMetric struct {
	value interface{}
	stats struct {
		min         float64
		max         float64
		sum         float64
		sampleCount int64
	}
}

const cloudWatchVersion = "2010-08-01"

func NewCloudWatchReporter(registry metrics.Registry, interval time.Duration, region, accessKey, secretKey, securityToken, namespace string, dimensions map[string]string, percentiles map[string]float64, timeout time.Duration) *PeriodicReporter {
	lr := newCloudWatchReporter(interval, region, accessKey, secretKey, securityToken, namespace, dimensions, percentiles, timeout)
	return NewPeriodicReporter(registry, interval, true, lr)
}

func newCloudWatchReporter(interval time.Duration, region, accessKey, secretKey, securityToken, namespace string, dimensions map[string]string, percentiles map[string]float64, timeout time.Duration) *cloudWatchReporter {
	per := metrics.DefaultPercentiles
	perNames := metrics.DefaultPercentileNames

	if percentiles != nil {
		per = make([]float64, 0)
		perNames = make([]string, 0)
		for name, p := range percentiles {
			per = append(per, p)
			perNames = append(perNames, name)
		}
	}

	if timeout == 0 {
		timeout = time.Second * 15
	}

	awsTransport := &http.Transport{
		Dial: (&net.Dialer{Timeout: timeout}).Dial,
		ResponseHeaderTimeout: timeout,
	}
	awsClient := &http.Client{
		Transport: awsTransport,
	}

	return &cloudWatchReporter{
		endpoint:        fmt.Sprintf("https://monitoring.%s.amazonaws.com", region),
		namespace:       namespace,
		percentiles:     per,
		percentileNames: perNames,
		counterCache:    &counterDeltaCache{},
		dimensions:      dimensions,
		securityToken:   securityToken,
		client: &aws4.Client{
			Keys: &aws4.Keys{
				AccessKey: accessKey,
				SecretKey: secretKey,
			},
			Client: awsClient,
		},
	}
}

func (r *cloudWatchReporter) Report(registry metrics.Registry) {
	mets := make(map[string]cloudWatchMetric)
	registry.Do(func(name string, metric interface{}) error {
		name = strings.Replace(name, "/", ".", -1)
		switch m := metric.(type) {
		case metrics.CounterValue:
			mets[name] = cloudWatchMetric{value: r.counterCache.delta(name, int64(m))}
		case metrics.GaugeValue:
			mets[name] = cloudWatchMetric{value: int64(m)}
		case metrics.IntegerGauge:
			mets[name] = cloudWatchMetric{value: int64(m.Value())}
		case metrics.Counter:
			mets[name] = cloudWatchMetric{value: r.counterCache.delta(name, m.Count())}
		case *metrics.EWMA:
			mets[name] = cloudWatchMetric{value: m.Rate()}
		case *metrics.EWMAGauge:
			mets[name] = cloudWatchMetric{value: m.Mean()}
		case *metrics.Meter:
			mets[name+".1m"] = cloudWatchMetric{value: m.OneMinuteRate()}
			mets[name+".5m"] = cloudWatchMetric{value: m.FiveMinuteRate()}
			mets[name+".15m"] = cloudWatchMetric{value: m.FifteenMinuteRate()}
		case metrics.Histogram:
			count := m.Count()
			if count > 0 {
				deltaCount := r.counterCache.delta(name+".count", int64(count))
				if deltaCount > 0 {
					deltaSum := r.counterCache.delta(name+".sum", m.Sum())
					w := cloudWatchMetric{}
					w.stats.max = float64(deltaSum) / float64(deltaCount)
					w.stats.min = float64(deltaSum) / float64(deltaCount)
					w.stats.sum = float64(deltaSum)
					w.stats.sampleCount = deltaCount
					mets[name] = w
				}
				percentiles := m.Percentiles(r.percentiles)
				for i, perc := range percentiles {
					mets[name+"."+r.percentileNames[i]] = cloudWatchMetric{value: perc}
				}
			}
		default:
			log.Printf("metrics/reporter/cloudwatch: unrecognized metric type for %s: %T %+v", name, m, m)
		}
		return nil
	})

	if len(mets) > 0 {
		// TODO: max POST size to CloudWatch is 40KB. Break up larger payloads over multiple requests.
		params := url.Values{}
		params.Set("Namespace", r.namespace)
		params.Set("Action", "PutMetricData")
		params.Set("Version", cloudWatchVersion)
		if r.securityToken != "" {
			params.Set("SecurityToken", r.securityToken)
		}
		idx := 1
		for name, m := range mets {
			prefix := fmt.Sprintf("MetricData.member.%d.", idx)
			if m.value != nil {
				switch x := m.value.(type) {
				case float64:
					params.Set(prefix+"Value", strconv.FormatFloat(x, 'E', 10, 64))
				case int64:
					params.Set(prefix+"Value", strconv.FormatInt(x, 10))
				case uint64:
					params.Set(prefix+"Value", strconv.FormatUint(x, 10))
				default:
					log.Printf("metrics/report/cloudwatch: unrecognized value type %T", m.value)
				}
			} else if m.stats.sampleCount > 0 {
				params.Set(prefix+"StatisticValues.Sum", strconv.FormatFloat(m.stats.sum, 'E', 10, 64))
				params.Set(prefix+"StatisticValues.SampleCount", strconv.FormatInt(m.stats.sampleCount, 10))
				params.Set(prefix+"StatisticValues.Minimum", strconv.FormatFloat(m.stats.min, 'E', 10, 64))
				params.Set(prefix+"StatisticValues.Maximum", strconv.FormatFloat(m.stats.max, 'E', 10, 64))
			} else {
				log.Printf("metrics/reporter/cloudwatch: metric %s missing value or statistics", name)
				continue
			}
			params.Set(prefix+"MetricName", name)
			dIdx := 0
			for name, value := range r.dimensions {
				dIdx++
				p := fmt.Sprintf("%sDimensions.member.%d.", prefix, dIdx)
				params.Set(p+"Name", name)
				params.Set(p+"Value", value)
			}
			idx++
		}
		res, err := r.client.PostForm(r.endpoint, params)
		if err != nil {
			log.Printf("metrics/reporter/cloudwatch: failed to send metrics to CloudWatch: %+v", err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Printf("metrics/reporter/cloudwatch: failed to read response body: %+v", err)
			} else {
				log.Printf("metrics/reporter/cloudwatch: failed to send metrics to CloudWatch: %d %s", res.StatusCode, string(body))
			}
		}
	}
}
