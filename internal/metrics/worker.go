package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	scoresWorkerRunsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scores_worker_runs_total",
			Help: "Total number of scores worker runs by result (success or error)",
		},
		[]string{"result"},
	)

	scoresWorkerLastSuccessTimestamp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "scores_worker_last_success_timestamp_seconds",
			Help: "Unix timestamp of the last successful scores worker run; alert when it grows stale",
		},
	)

	scoresRecordedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "scores_recorded_total",
			Help: "Total number of new quarter scores recorded from the scoreboard",
		},
	)
)

func init() {
	prometheus.MustRegister(
		scoresWorkerRunsTotal,
		scoresWorkerLastSuccessTimestamp,
		scoresRecordedTotal,
	)
}

func IncScoresRun(success bool) {
	if success {
		scoresWorkerRunsTotal.WithLabelValues("success").Inc()
		scoresWorkerLastSuccessTimestamp.Set(float64(time.Now().Unix()))
	} else {
		scoresWorkerRunsTotal.WithLabelValues("error").Inc()
	}
}

func AddScoresRecorded(n int) {
	scoresRecordedTotal.Add(float64(n))
}
