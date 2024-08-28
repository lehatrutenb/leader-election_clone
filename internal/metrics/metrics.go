package metrics

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"
)

type Metrics struct {
	AmtStateChanges   prometheus.Counter
	CurState          prometheus.Gauge
	CurStateStartTime prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		AmtStateChanges: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "amt_state_changes",
			Help: "Amount of states changes.",
		}),
		CurState: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "cur_state",
				Help: "Numeric name of current running state.",
			},
		),
		CurStateStartTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "cur_state_start_time",
				Help: "Start time of current running state.",
			},
		),
	}

	reg.MustRegister(m.AmtStateChanges)
	reg.MustRegister(m.CurState)
	reg.MustRegister(m.CurStateStartTime)

	return m
}

func InitPrometheus(ctx context.Context, logger *slog.Logger, eg *errgroup.Group) *Metrics {
	logger = logger.With("subsystem", "Prometheus")
	logger.LogAttrs(ctx, slog.LevelInfo, "Start initializing prometheus")
	reg := prometheus.NewRegistry()

	m := newMetrics(reg)

	srv := &http.Server{Addr: ":8080"}
	eg.Go(func() error {
		<-ctx.Done()
		logger.LogAttrs(ctx, slog.LevelInfo, "Prometheus http server is shutting down")
		return srv.Shutdown(ctx)
	})

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	eg.Go(func() error {
		if err := srv.ListenAndServe(); err != nil {
			logger.LogAttrs(ctx, slog.LevelError, "Prometheus http server fell down: "+err.Error())
			return err
		}
		return nil
	})

	return m
}
