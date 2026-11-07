package main

import (
	"net/http"

	"omada_exporter_go/internal"
	"omada_exporter_go/internal/Log"
	"omada_exporter_go/internal/Prometheus"
)

func main() {
	Log.Init()
	Log.Info("Starting %s", internal.AppName)
	http.Handle("/metrics", Prometheus.OmadaMetricsHandler())
	http.Handle("/metrics/clients", Prometheus.ClientMetricsHandler())
	Prometheus.StartClientPoller()
	if err := http.ListenAndServe(":"+internal.GetConfig().Prometheus.MetricsPort, nil); err != nil {
		panic(err)
	}
}
