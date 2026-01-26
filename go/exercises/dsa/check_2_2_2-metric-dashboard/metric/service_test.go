package metric_test

import (
	"testing"
	"time"

	"check_2_2_2-metric-dashboard/metric"
)

func TestAggregator(t *testing.T) {
	agg := metric.NewAggregator()
	metricID := "cpu.usage"
	agg.Add(metricID, 0.75)
	agg.Add(metricID, 0.35)
	agg.Add(metricID, 0.5)
	agg.Add(metricID, 0.4)
	agg.Add(metricID, 0.7)
	agg.Add(metricID, 0.9)

	stats := agg.Stats(metricID, time.Now())
	t.Logf("Stats: %+v", stats)
}
