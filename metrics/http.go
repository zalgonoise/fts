package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func newServer(port int, registry *prometheus.Registry) *http.Server {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		Registry:          registry,
		EnableOpenMetrics: true,
	}))

	server := &http.Server{
		Handler:      mux,
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	return server
}
