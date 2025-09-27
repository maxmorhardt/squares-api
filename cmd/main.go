package main

import (
	"net/http"
	"squares-api/internal/routes"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	go prometheus()
	gin()
}

func prometheus() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}

func gin() {
	r := routes.SetupRouter()
	r.Run(":8080")
}