package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type Config struct {
	ESPNBaseURL      string
	PollInterval     time.Duration
	ScheduleInterval time.Duration
	LockKey          int64
}

type Runner struct {
	db               *gorm.DB
	poller           *poller
	scheduler        *scheduler
	pollInterval     time.Duration
	scheduleInterval time.Duration
	lockKey          int64
	log              *slog.Logger
}

func NewRunner(db *gorm.DB, gameRepo repository.GameRepository, gameService service.GameService, cfg Config, log *slog.Logger) *Runner {
	espn := newESPNClient(cfg.ESPNBaseURL)
	return &Runner{
		db:               db,
		poller:           newPoller(espn, gameRepo, gameService),
		scheduler:        newScheduler(espn, gameRepo),
		pollInterval:     cfg.PollInterval,
		scheduleInterval: cfg.ScheduleInterval,
		lockKey:          cfg.LockKey,
		log:              log,
	}
}

func (r *Runner) Start(ctx context.Context) {
	go r.loop(ctx, "schedule", r.scheduleInterval, r.lockKey, r.scheduler.sync)
	go r.loop(ctx, "poll", r.pollInterval, r.lockKey+1, r.poller.poll)
}

func (r *Runner) loop(ctx context.Context, name string, interval time.Duration, lockKey int64, fn func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// run once at startup so a fresh deploy syncs/polls without waiting a full interval
	r.runGuarded(ctx, name, lockKey, fn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runGuarded(ctx, name, lockKey, fn)
		}
	}
}

func (r *Runner) runGuarded(ctx context.Context, name string, lockKey int64, fn func(context.Context) error) {
	sqlDB, err := r.db.DB()
	if err != nil {
		r.log.Error("failed to get sql db for advisory lock", "job", name, "error", err)
		return
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		r.log.Error("failed to acquire connection for advisory lock", "job", name, "error", err)
		return
	}
	defer func() { _ = conn.Close() }()

	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&locked); err != nil {
		r.log.Error("failed to acquire advisory lock", "job", name, "error", err)
		return
	}
	if !locked {
		return
	}
	defer func() {
		// unlock on a fresh context so shutdown cancellation can't leave it held
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
			r.log.Error("failed to release advisory lock", "job", name, "error", err)
		}
	}()

	jobCtx := util.ContextWithLogger(ctx, r.log.With("job", name))
	if err := fn(jobCtx); err != nil {
		r.log.Error("scores job failed", "job", name, "error", err)
	}
}
