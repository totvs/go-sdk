package handlers

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func HealthLive(w http.ResponseWriter, r *http.Request) {
	log.Printf("Health liveness check accessed from %s", r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func HealthReady(w http.ResponseWriter, r *http.Request) {
	log.Printf("Health readiness check accessed from %s", r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func Metrics(registry *prometheus.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Metrics endpoint accessed from %s", r.RemoteAddr)
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	}
}
