package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	PagesFetched = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "crawler_pages_fetched_total",
		Help: "Total pages fetched successfully",
	})
	FetchErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "crawler_fetch_errors_total",
		Help: "Total fetch errors by class",
	}, []string{"class"})
	QueueDepth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "crawler_queue_depth",
		Help: "Queue depth by stage",
	}, []string{"stage"})
)

func init() {
	prometheus.MustRegister(PagesFetched, FetchErrors, QueueDepth)
}