package collectors

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/frodenas/firehose_exporter/metrics"
	"github.com/frodenas/firehose_exporter/utils"
)

var (
	valueMetricsCollectorDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, value_metrics_subsystem, "collector"),
		"Cloud Foundry firehose value metrics collector.",
		nil,
		nil,
	)
)

type valueMetricsCollector struct {
	metricsStore *metrics.Store
}

func NewValueMetricsCollector(metricsStore *metrics.Store) *valueMetricsCollector {
	collector := &valueMetricsCollector{
		metricsStore: metricsStore,
	}
	return collector
}

func (c valueMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, valueMetric := range c.metricsStore.GetValueMetrics() {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, value_metrics_subsystem, utils.NormalizeName(valueMetric.Name)),
				fmt.Sprintf("Cloud Foundry firehose '%s' value metric.", valueMetric.Name),
				[]string{"origin", "deployment", "job", "index", "ip", "unit"},
				nil,
			),
			prometheus.GaugeValue,
			float64(valueMetric.Value),
			valueMetric.Origin,
			valueMetric.Deployment,
			valueMetric.Job,
			valueMetric.Index,
			valueMetric.IP,
			valueMetric.Unit,
		)
	}
}

func (c valueMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- valueMetricsCollectorDesc
}
