package metric

import (
	"stl/deque"
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

// WithRetentionWindow sets the retention window.
func WithRetentionWindow(window time.Duration) AggregatorOption {
	return func(c *AggregatorConfig) { c.RetentionWindow = window }
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

// - Sliding window aggregation

type SlidingAggregator struct {
	config  *SlidingAggregatorConfig
	metrics map[string]*slidingWindowMetric
}

// NewSlidingAggregator returns a new SlidingAggregator with a default configuration.
func NewSlidingAggregator(opts ...SlidingAggregatorOption) *SlidingAggregator {
	options := defaultSlidingAggregatorConfig()
	for _, opt := range opts {
		opt(options)
	}
	return NewSlidingAggregatorWithConfig(options)
}

// NewSlidingAggregatorWithConfig returns a new SlidingAggregator with the given configuration.
func NewSlidingAggregatorWithConfig(config *SlidingAggregatorConfig) *SlidingAggregator {
	return &SlidingAggregator{config: config, metrics: make(map[string]*slidingWindowMetric)}
}

type slidingWindowMetric struct {
	data  *deque.Deque[slidingWindowData]
	sum   float64
	count int64
}

type slidingWindowData struct {
	timestamp time.Time
	value     float64
}

func (a *SlidingAggregator) Add(metricID string, value float64, timestamp time.Time) {
	metric, ok := a.metrics[metricID]
	if !ok {
		metric = &slidingWindowMetric{data: deque.NewDeque[slidingWindowData]()}
		a.metrics[metricID] = metric
	}
	metric.data.PushBack(slidingWindowData{timestamp: timestamp, value: value})
	metric.sum += value
	metric.count++
}

func (a *SlidingAggregator) Average(metricID string, now time.Time) float64 {
	metric, ok := a.metrics[metricID]
	if !ok {
		return 0
	}

	cutoff := now.Add(-a.config.RetentionWindow)
	for !metric.data.Empty() {
		item := metric.data.Front()
		if item.timestamp.Before(cutoff) {
			metric.data.PopFront()
			metric.sum -= item.value
			metric.count--
		} else {
			break // Remaining entries are newer, stop evicting
		}
	}
	if metric.count == 0 {
		return 0
	}
	return metric.sum / float64(metric.count)
}

// -- Options

type SlidingAggregatorConfig struct {
	RetentionWindow time.Duration
}

func defaultSlidingAggregatorConfig() *SlidingAggregatorConfig {
	return &SlidingAggregatorConfig{RetentionWindow: time.Minute * 5}
}

type SlidingAggregatorOption func(aggregator *SlidingAggregatorConfig)

// WithSlidingRetentionWindow sets the retention window.
func WithSlidingRetentionWindow(window time.Duration) SlidingAggregatorOption {
	return func(c *SlidingAggregatorConfig) { c.RetentionWindow = window }
}
