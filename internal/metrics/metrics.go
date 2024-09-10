package metrics

import (
	cadre_metrics "github.com/moderntv/cadre/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricsPrefix = "cache_"
	subSystem     = "codebook_cache"
	labelName     = "name"
)

type Metrics struct {
	ItemsCount                prometheus.Gauge
	LoadCount                 prometheus.Counter
	ReloadInterval            prometheus.Gauge
	ReceivedNatsInvalidations prometheus.Counter
	MemoryUsage               prometheus.Gauge
}

func New(
	name string,
	registry *cadre_metrics.Registry,
) (m *Metrics, err error) {
	itemsCount := registry.NewGauge(prometheus.GaugeOpts{
		Subsystem:   subSystem,
		Name:        "items_count",
		Help:        "Count of cached items",
		ConstLabels: prometheus.Labels{labelName: name},
	})

	loadCount := registry.NewCounter(prometheus.CounterOpts{
		Subsystem:   subSystem,
		Name:        "load_count",
		Help:        "Total number of complete loads",
		ConstLabels: prometheus.Labels{labelName: name},
	})

	receivedNatsInvalidations := registry.NewCounter(prometheus.CounterOpts{
		Subsystem:   subSystem,
		Name:        "received_nats_invalidations",
		Help:        "Total number of received invalidations",
		ConstLabels: prometheus.Labels{labelName: name},
	})

	memoryUsage := registry.NewGauge(prometheus.GaugeOpts{
		Subsystem:   subSystem,
		Name:        "memory_usage",
		Help:        "Current memory usage in bytes by entries",
		ConstLabels: prometheus.Labels{labelName: name},
	})

	err = registry.Register(metricsPrefix+name+"_items_count", itemsCount)
	if err != nil {
		return
	}

	err = registry.Register(metricsPrefix+name+"_load_count", loadCount)
	if err != nil {
		return
	}

	err = registry.Register(metricsPrefix+name+"_received_nats_invalidations", receivedNatsInvalidations)
	if err != nil {
		return
	}

	err = registry.Register(metricsPrefix+name+"_memory_usage", memoryUsage)
	if err != nil {
		return
	}

	m = &Metrics{
		ItemsCount:                itemsCount,
		LoadCount:                 loadCount,
		ReceivedNatsInvalidations: receivedNatsInvalidations,
		MemoryUsage:               memoryUsage,
	}

	return
}
