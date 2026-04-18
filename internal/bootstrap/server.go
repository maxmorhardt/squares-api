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
)

func NewServer() *gin.Engine {
	config.LoadEnv()

	config.InitOIDC()
	config.InitDB()
	config.InitNATS()

	r := gin.New()

	setupMiddleware(r)
	setupRoutes(r)
	setupMetricsServer()
	setupValidators()

	return r
}

func setupMiddleware(r *gin.Engine) {
	r.Use(gin.Recovery())
	r.Use(middleware.RequestSizeLimitMiddleware())
	r.Use(middleware.CORSMiddleware(config.Env().Server.AllowedOrigins))
	r.Use(middleware.LoggerMiddleware)

	if config.Env().Server.MetricsEnabled {
		r.Use(middleware.PrometheusMiddleware)
	}
}

func setupRoutes(r *gin.Engine) {
	db := config.DB()

	contestRepo := repository.NewContestRepository(db)
	contactRepo := repository.NewContactRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	participantRepo := repository.NewParticipantRepository(db)

	authService := service.NewAuthService()
	natsService := service.NewNatsService()
	participantService := service.NewParticipantService(participantRepo, contestRepo, natsService)
	contestService := service.NewContestService(contestRepo, participantRepo, natsService, authService, participantService)
	wsService := service.NewWebSocketService()
	contactService := service.NewContactService(contactRepo)
	inviteService := service.NewInviteService(inviteRepo, participantRepo, contestRepo, participantService, natsService)

	statsRepo := repository.NewStatsRepository(db)
	statsService := service.NewStatsService(statsRepo)

	contestHandler := handler.NewContestHandler(contestService, authService)
	wsHandler := handler.NewWebSocketHandler(wsService, contestRepo, participantService)
	contactHandler := handler.NewContactHandler(contactService)
	statsHandler := handler.NewStatsHandler(statsService)
	inviteHandler := handler.NewInviteHandler(inviteService)
	participantHandler := handler.NewParticipantHandler(participantService)
	healthHandler := handler.NewHealthHandler(db, config.NATS, config.OIDCVerifier)

	routes.RegisterRootRoutes(r.Group(""), healthHandler)
	routes.RegisterStatsRoutes(r.Group("/stats"), statsHandler)
	routes.RegisterContactRoute(r.Group("/contact"), contactHandler)
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler)

	routes.RegisterInviteRoutes(r.Group("/invites"), inviteHandler)
	routes.RegisterContestInviteRoutes(r.Group("/contests/:id/invites"), inviteHandler)

	routes.RegisterMyContestsRoute(r.Group("/contests/me"), participantHandler)
	routes.RegisterParticipantRoutes(r.Group("/contests/:id/participants"), participantHandler)
}
