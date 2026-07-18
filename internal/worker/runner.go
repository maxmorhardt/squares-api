package worker

import (
	"context"
	"time"

	"github.com/maxmorhardt/squares-api/internal/clients"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type Runner interface {
	Start(ctx context.Context)
}

type runner struct {
	db      *gorm.DB
	worker  *scoresWorker
	lockKey int64
}

func NewRunner(db *gorm.DB, gameService service.GameService, cfg model.WorkerConfig) Runner {
	espn := clients.NewESPNClient(cfg.ESPNBaseURL)
	return &runner{
		db:      db,
		worker:  newScoresWorker(espn, gameService, cfg.ActiveInterval, cfg.IdleInterval),
		lockKey: cfg.LockKey,
	}
}

func (r *runner) Start(ctx context.Context) {
	ctx = util.ContextWithLogger(ctx, util.LoggerFromContext(ctx).With("job", "scores"))
	go r.loop(ctx)
}

func (r *runner) loop(ctx context.Context) {
	// run once at startup so a fresh deploy syncs without waiting a full interval
	r.runGuarded(ctx)

	for {
		delay := r.worker.nextDelay(ctx)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
			r.runGuarded(ctx)
		}
	}
}

func (r *runner) runGuarded(ctx context.Context) {
	log := util.LoggerFromContext(ctx)

	// pin a single connection so the advisory lock lives on one session
	sqlDB, err := r.db.DB()
	if err != nil {
		log.Error("failed to get sql db for advisory lock", "error", err)
		return
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		log.Error("failed to acquire connection for advisory lock", "error", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// only one replica should poll ESPN at a time
	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", r.lockKey).Scan(&locked); err != nil {
		log.Error("failed to acquire advisory lock", "error", err)
		return
	}
	// another replica holds the lock, so skip this turn
	if !locked {
		return
	}
	defer func() {
		// unlock on a fresh context so shutdown cancellation can't leave it held
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", r.lockKey); err != nil {
			log.Error("failed to release advisory lock", "error", err)
		}
	}()

	// record the outcome so an alert can fire when the worker stops making progress
	if err := r.worker.run(ctx); err != nil {
		log.Error("scores job failed", "error", err)
		metrics.IncScoresRun(false)
		return
	}
	metrics.IncScoresRun(true)
}
