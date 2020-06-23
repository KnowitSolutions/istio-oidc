package introspection

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func init() {
	metrics.Registry.Unregister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	metrics.Registry.Unregister(prometheus.NewGoCollector())
}

func RegisterMetrics(mux *http.ServeMux) {
	gaths := prometheus.Gatherers{prometheus.DefaultGatherer, metrics.Registry}
	mux.Handle("/metrics", promhttp.HandlerFor(gaths, promhttp.HandlerOpts{}))
}
