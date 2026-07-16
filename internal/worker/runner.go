package worker

import (
	"context"
	"time"

	"github.com/maxmorhardt/squares-api/internal/clients"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type jobName string

const (
	jobSchedule jobName = "schedule"
	jobPoll     jobName = "poll"
)

type Runner interface {
	Start(ctx context.Context)
}

type runner struct {
	db               *gorm.DB
	poller           *poller
	scheduler        *scheduler
	pollInterval     time.Duration
	scheduleInterval time.Duration
	lockKey          int64
}

func NewRunner(db *gorm.DB, gameRepo repository.GameRepository, gameService service.GameService, cfg model.WorkerConfig) Runner {
	espn := clients.NewESPNClient(cfg.ESPNBaseURL)
	return &runner{
		db:               db,
		poller:           newPoller(espn, gameRepo, gameService),
		scheduler:        newScheduler(espn, gameRepo),
		pollInterval:     cfg.PollInterval,
		scheduleInterval: cfg.ScheduleInterval,
		lockKey:          cfg.LockKey,
	}
}

func (r *runner) Start(ctx context.Context) {
	go r.loop(ctx, jobSchedule, r.scheduleInterval, r.lockKey, r.scheduler.sync)
	go r.loop(ctx, jobPoll, r.pollInterval, r.lockKey+1, r.poller.poll)
}

func (r *runner) loop(ctx context.Context, name jobName, interval time.Duration, lockKey int64, fn func(context.Context) error) {
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

func (r *runner) runGuarded(ctx context.Context, name jobName, lockKey int64, fn func(context.Context) error) {
	log := util.LoggerFromContext(ctx).With("job", string(name))

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

	var locked bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&locked); err != nil {
		log.Error("failed to acquire advisory lock", "error", err)
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
			log.Error("failed to release advisory lock", "error", err)
		}
	}()

	if err := fn(util.ContextWithLogger(ctx, log)); err != nil {
		log.Error("scores job failed", "error", err)
	}
}
