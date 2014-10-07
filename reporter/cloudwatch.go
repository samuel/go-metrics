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
	namespace  string
	client     *aws4.Client
	dimensions map[string]string
	endpoint   string
	authFunc   AWSAuthFunc
}

type cloudWatchMetric struct {
	value interface{}
	stats struct {
		min         float64
		max         float64
		sum         float64
		sampleCount uint64
	}
}

const cloudWatchVersion = "2010-08-01"

type AWSAuthFunc func() (accessKey string, secretKey string, securityToken string)

func NewCloudWatchReporter(registry metrics.Registry, interval time.Duration, latched bool, region string, authFunc AWSAuthFunc, namespace string, dimensions map[string]string, timeout time.Duration) *PeriodicReporter {
	lr := newCloudWatchReporter(interval, region, authFunc, namespace, dimensions, timeout)
	return NewPeriodicReporter(registry, interval, true, latched, lr)
}

func newCloudWatchReporter(interval time.Duration, region string, authFunc AWSAuthFunc, namespace string, dimensions map[string]string, timeout time.Duration) *cloudWatchReporter {
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
		endpoint:   fmt.Sprintf("https://monitoring.%s.amazonaws.com", region),
		namespace:  namespace,
		dimensions: dimensions,
		authFunc:   authFunc,
		client: &aws4.Client{
			Keys:   &aws4.Keys{},
			Client: awsClient,
		},
	}
}

func (r *cloudWatchReporter) Report(snapshot *metrics.RegistrySnapshot) {
	mets := make(map[string]cloudWatchMetric)

	for _, v := range snapshot.Values {
		mets[strings.Replace(v.Name, "/", ".", -1)] = cloudWatchMetric{value: v.Value}
	}
	for _, v := range snapshot.Distributions {
		m := cloudWatchMetric{}
		m.stats.min = v.Value.Min
		m.stats.max = v.Value.Max
		m.stats.sum = v.Value.Sum
		m.stats.sampleCount = v.Value.Count
		mets[strings.Replace(v.Name, "/", ".", -1)] = m
	}

	if len(mets) > 0 {
		// TODO: max POST size to CloudWatch is 40KB. Break up larger payloads over multiple requests.
		params := url.Values{}
		params.Set("Namespace", r.namespace)
		params.Set("Action", "PutMetricData")
		params.Set("Version", cloudWatchVersion)
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
				params.Set(prefix+"StatisticValues.SampleCount", strconv.FormatUint(m.stats.sampleCount, 10))
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
		accessKey, secretKey, securityToken := r.authFunc()
		r.client.Keys.AccessKey = accessKey
		r.client.Keys.SecretKey = secretKey
		if securityToken != "" {
			params.Set("SecurityToken", securityToken)
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
