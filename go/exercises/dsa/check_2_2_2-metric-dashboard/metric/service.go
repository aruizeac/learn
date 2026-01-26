package metric

import (
	"time"
)

// - Aggregator

type Aggregator struct {
	config     *AggregatorConfig
	metricData map[string]*metricData
}

type metricData struct {
	buckets [30]bucket // a circular buffer (buckets) per metric
}

type bucket struct {
	windowStart time.Time
	min         float64
	max         float64
	sum         float64
	count       int64
}

type Stats struct {
	Average float64
	Min     float64
	Max     float64
	Count   int64
	Sum     float64
}

// NewAggregatorWithConfig returns a new Aggregator with the given configuration.
func NewAggregatorWithConfig(config *AggregatorConfig) *Aggregator {
	return &Aggregator{config: config, metricData: make(map[string]*metricData)}
}

// NewAggregator returns a new Aggregator with a default configuration.
func NewAggregator(opts ...AggregatorOption) *Aggregator {
	options := defaultAggregatorConfig()
	for _, opt := range opts {
		opt(options)
	}
	return NewAggregatorWithConfig(options)
}

// Add adds a metric to the aggregator.
func (a *Aggregator) Add(metricID string, value float64, opts ...AddOption) {
	metricOpts := defaultAddMetricOptions()
	for _, opt := range opts {
		opt(metricOpts)
	}

	data, ok := a.metricData[metricID]
	if !ok {
		data = &metricData{buckets: [30]bucket{}}
		a.metricData[metricID] = data
	}

	bucketIdx := bucketIndex(metricOpts.timestamp, a.config.BucketWindowSize)
	expWindow := metricOpts.timestamp.Truncate(a.config.BucketWindowSize)
	if data.buckets[bucketIdx].windowStart.Equal(expWindow) {
		// within the same bucket window
		data.buckets[bucketIdx].sum += value
		data.buckets[bucketIdx].count++
		data.buckets[bucketIdx].min = min(data.buckets[bucketIdx].min, value)
		data.buckets[bucketIdx].max = max(data.buckets[bucketIdx].max, value)
		return
	}
	// reset stale bucket
	data.buckets[bucketIdx] = bucket{
		windowStart: expWindow,
		min:         value,
		max:         value,
		sum:         value,
		count:       1,
	}
}

func (a *Aggregator) Average(metricID string, now time.Time) float64 {
	data, ok := a.metricData[metricID]
	if !ok {
		return 0
	}

	totalSum := float64(0)
	totalCount := int64(0)
	cutoff := now.Add(-a.config.RetentionWindow)
	for _, b := range data.buckets {
		if b.windowStart.Before(cutoff) || b.windowStart.After(now) {
			continue // outside retention window
		}
		totalSum += b.sum
		totalCount += b.count
	}
	if totalCount == 0 {
		return 0
	}
	return totalSum / float64(totalCount)
}

func (a *Aggregator) Stats(metricID string, now time.Time) Stats {
	data, ok := a.metricData[metricID]
	if !ok {
		return Stats{}
	}

	stats := Stats{}
	cutoff := now.Add(-a.config.RetentionWindow)
	firstBucket := true
	for _, b := range data.buckets {
		if b.windowStart.Before(cutoff) || b.windowStart.After(now) {
			continue // outside retention window
		}

		if firstBucket {
			stats.Min = b.min
			stats.Max = b.max
			firstBucket = false
		} else {
			stats.Min = min(stats.Min, b.min)
			stats.Max = max(stats.Max, b.max)
		}
		stats.Count += b.count
		stats.Sum += b.sum
	}
	if stats.Count > 0 {
		stats.Average = stats.Sum / float64(stats.Count)
	}
	return stats
}

// -- Options

type AggregatorConfig struct {
	BucketWindowSize time.Duration
	RetentionWindow  time.Duration
}

func defaultAggregatorConfig() *AggregatorConfig {
	return &AggregatorConfig{BucketWindowSize: time.Second * 10, RetentionWindow: time.Minute * 5}
}

type AggregatorOption func(*AggregatorConfig)

// WithBucketWindowSize sets the bucket window size.
func WithBucketWindowSize(size time.Duration) AggregatorOption {
	return func(c *AggregatorConfig) { c.BucketWindowSize = size }
}

type addMetricOptions struct {
	timestamp time.Time
}

func defaultAddMetricOptions() *addMetricOptions {
	return &addMetricOptions{timestamp: time.Now()}
}

type AddOption func(*addMetricOptions)

// WithTimestamp sets the timestamp of the metric.
func WithTimestamp(t time.Time) AddOption {
	return func(o *addMetricOptions) { o.timestamp = t }
}

// -- Utils

// bucketIndex returns the index of the bucket for the given timestamp.
//
// It enables even distribution of metrics across buckets.
func bucketIndex(timestamp time.Time, bucketSize time.Duration) int {
	return int(timestamp.Unix()/int64(bucketSize.Seconds())) % 30
}
