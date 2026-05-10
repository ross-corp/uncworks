package server

import (
	"context"
	"expvar"

	"github.com/prometheus/client_golang/prometheus"
	aotv1alpha1 "github.com/uncworks/aot/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// RunsCreatedTotal counts the total number of agent runs created.
	RunsCreatedTotal = expvar.NewInt("uncworks_runs_created_total")
	// RunsSucceededTotal counts the total number of agent runs that succeeded.
	RunsSucceededTotal = expvar.NewInt("uncworks_runs_succeeded_total")
	// RunsFailedTotal counts the total number of agent runs that failed.
	RunsFailedTotal = expvar.NewInt("uncworks_runs_failed_total")
	// RunsCancelledTotal counts the total number of agent runs that were cancelled.
	RunsCancelledTotal = expvar.NewInt("uncworks_runs_cancelled_total")
)

var knownPhases = []string{"Pending", "Running", "WaitingForInput", "Succeeded", "Failed", "Cancelled"}

// MetricsCollector is a Prometheus Collector that reports live AgentRun phase counts.
type MetricsCollector struct {
	k8sClient client.Client
	namespace string
	desc      *prometheus.Desc
}

// NewMetricsCollector creates a MetricsCollector using the provided k8s client.
func NewMetricsCollector(k8sClient client.Client, namespace string) *MetricsCollector {
	return &MetricsCollector{
		k8sClient: k8sClient,
		namespace: namespace,
		desc: prometheus.NewDesc(
			"aot_agent_runs_total",
			"Number of AgentRun objects by phase.",
			[]string{"phase"},
			nil,
		),
	}
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	list := &aotv1alpha1.AgentRunList{}
	if err := c.k8sClient.List(context.Background(), list, client.InNamespace(c.namespace)); err != nil {
		return
	}

	counts := make(map[string]float64, len(knownPhases))
	for _, phase := range knownPhases {
		counts[phase] = 0
	}
	for _, run := range list.Items {
		phase := string(run.Status.Phase)
		if phase == "" {
			phase = "Pending"
		}
		counts[phase]++
	}

	for _, phase := range knownPhases {
		ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, counts[phase], phase)
	}
}