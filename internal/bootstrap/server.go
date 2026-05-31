package bootstrap

import (
	"github.com/coreos/go-oidc/v3/oidc"
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
	Config *config.Config
	DB     *gorm.DB
	NATS   *nats.Conn
	OIDC   *oidc.IDTokenVerifier
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
		return nil, err
	}

	verifier, err := config.InitOIDC(cfg)
	if err != nil {
		return nil, err
	}

	return &Dependencies{
		Config: cfg,
		DB:     db,
		NATS:   nc,
		OIDC:   verifier,
	}, nil
}

func NewServer(deps *Dependencies) *gin.Engine {
	r := gin.New()

	setupMiddleware(r, deps.Config)
	setupRoutes(r, deps)
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

func setupRoutes(r *gin.Engine, deps *Dependencies) {
	db := deps.DB
	verifier := middleware.NewOIDCTokenVerifier(deps.OIDC)

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

	contestHandler := handler.NewContestHandler(contestService)
	wsHandler := handler.NewWebSocketHandler(wsService, contestRepo, participantService, deps.Config.Server.AllowedOrigins, deps.NATS)
	contactHandler := handler.NewContactHandler(contactService)
	statsHandler := handler.NewStatsHandler(statsService)
	inviteHandler := handler.NewInviteHandler(inviteService)
	participantHandler := handler.NewParticipantHandler(participantService)
	healthHandler := handler.NewHealthHandler(db, deps.NATS, deps.OIDC)

	routes.RegisterRootRoutes(r.Group(""), healthHandler)
	routes.RegisterStatsRoutes(r.Group("/stats"), statsHandler)
	routes.RegisterContactRoute(r.Group("/contact"), contactHandler, deps.Config.Server.ContactRateLimit)
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler, verifier)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler, verifier)

	routes.RegisterInviteRoutes(r.Group("/invites"), inviteHandler, verifier)
	routes.RegisterContestInviteRoutes(r.Group("/contests/:id/invites"), inviteHandler, verifier)

	routes.RegisterMyContestsRoute(r.Group("/contests/me"), participantHandler, verifier)
	routes.RegisterParticipantRoutes(r.Group("/contests/:id/participants"), participantHandler, verifier)
}
