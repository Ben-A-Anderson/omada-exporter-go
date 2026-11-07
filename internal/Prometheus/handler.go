package Prometheus

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collections are serialised and cached so overlapping scrapes can't stack up.
const minCollectInterval = 50 * time.Second

var (
	collectMu   sync.Mutex
	lastCollect time.Time
)

func OmadaMetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		collectMu.Lock()
		if time.Since(lastCollect) >= minCollectInterval {
			CollectMetrics()
			lastCollect = time.Now()
		}
		collectMu.Unlock()
		promhttp.Handler().ServeHTTP(w, r)
	})
}
