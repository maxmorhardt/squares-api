package bootstrap

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/routes"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config   *config.Config
	DB       *gorm.DB
	NATS     *nats.Conn
	Verifier middleware.TokenVerifier
}

func BuildDependencies() (*Dependencies, error) {
	cfg, err := config.LoadEnv()
	if err != nil {
		return nil, err
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		return nil, err
	}

	nc, err := config.InitNATS(cfg)
	if err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	oidcVerifier, err := config.InitOIDC(cfg)
	if err != nil {
		nc.Close()
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	return &Dependencies{
		Config:   cfg,
		DB:       db,
		NATS:     nc,
		Verifier: middleware.NewOIDCTokenVerifier(oidcVerifier),
	}, nil
}

func NewServer(deps *Dependencies) *gin.Engine {
	r := gin.New()

	setupMiddleware(r, deps.Config)
	setupRoutes(r, deps, deps.Verifier)
	setupMetricsServer(deps.Config)
	setupValidators()

	return r
}

func setupMiddleware(r *gin.Engine, cfg *config.Config) {
	r.Use(gin.Recovery())
	r.Use(middleware.RequestSizeLimitMiddleware())
	r.Use(middleware.CORSMiddleware(cfg.Server.AllowedOrigins))
	r.Use(middleware.LoggerMiddleware)

	if cfg.Server.MetricsEnabled {
		r.Use(middleware.PrometheusMiddleware)
	}
}

func setupRoutes(r *gin.Engine, deps *Dependencies, verifier middleware.TokenVerifier) {
	db := deps.DB

	contestRepo := repository.NewContestRepository(db)
	contactRepo := repository.NewContactRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	participantRepo := repository.NewParticipantRepository(db)

	natsService := service.NewNatsService(deps.NATS)
	participantService := service.NewParticipantService(participantRepo, contestRepo, natsService)
	contestService := service.NewContestService(contestRepo, participantRepo, natsService, participantService)
	wsService := service.NewWebSocketService(deps.NATS)
	contactService := service.NewContactService(contactRepo, deps.Config)
	inviteService := service.NewInviteService(inviteRepo, participantRepo, contestRepo, participantService, natsService)

	statsRepo := repository.NewStatsRepository(db)
	statsService := service.NewStatsService(statsRepo)

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo, contestService)

	contestHandler := handler.NewContestHandler(contestService)
	wsHandler := handler.NewWebSocketHandler(wsService, contestRepo, participantService, deps.Config.Server.AllowedOrigins, deps.NATS)
	contactHandler := handler.NewContactHandler(contactService)
	statsHandler := handler.NewStatsHandler(statsService)
	inviteHandler := handler.NewInviteHandler(inviteService)
	participantHandler := handler.NewParticipantHandler(participantService)
	userHandler := handler.NewUserHandler(userService)
	healthHandler := handler.NewHealthHandler(db, deps.NATS, deps.Verifier)

	routes.RegisterRootRoutes(r.Group(""), healthHandler)
	routes.RegisterStatsRoutes(r.Group("/stats"), statsHandler)
	routes.RegisterContactRoute(r.Group("/contact"), contactHandler, deps.Config.Server.ContactRateLimit)
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler, verifier)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler, verifier)

	routes.RegisterInviteRoutes(r.Group("/invites"), inviteHandler, verifier)
	routes.RegisterContestInviteRoutes(r.Group("/contests/:id/invites"), inviteHandler, verifier)

	routes.RegisterMyContestsRoute(r.Group("/contests/me"), participantHandler, verifier)
	routes.RegisterParticipantRoutes(r.Group("/contests/:id/participants"), participantHandler, verifier)

	routes.RegisterUserRoutes(r.Group("/users/me"), userHandler, verifier)
}
