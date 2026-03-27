package metrics

import "github.com/prometheus/client_golang/prometheus"

var ActiveStreams = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "sprut",
	Name:      "active_streams",
	Help:      "Number of active SSE streams to MCP servers",
})

var ToolsFetchedTotal = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: "sprut",
	Name:      "tools_fetched_total",
	Help:      "Total number of tool list fetches from MCP servers",
})

func init() {
	prometheus.MustRegister(ActiveStreams, ToolsFetchedTotal)
}
