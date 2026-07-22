package bootstrap

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/routes"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func NewServer(deps *Dependencies) *gin.Engine {
	r := gin.New()

	setupMiddleware(r, deps.Config)
	setupRoutes(r, deps)
	setupMetricsServer(deps.Config)
	setupValidators()

	return r
}

func setupMiddleware(r *gin.Engine, cfg *model.AppConfig) {
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

	contestRepo := repository.NewContestRepository(db)
	contactRepo := repository.NewContactRepository(db)
	inviteRepo := repository.NewInviteRepository(db)
	participantRepo := repository.NewParticipantRepository(db)
	gameRepo := repository.NewGameRepository(db)

	userRepo := repository.NewUserRepository(db)

	natsService := service.NewNatsService(deps.NATS)
	userService := service.NewUserService(userRepo, natsService, deps.OIDCVerifier)

	participantService := service.NewParticipantService(participantRepo, contestRepo, natsService)
	contestService := service.NewContestService(contestRepo, participantRepo, gameRepo, userRepo, natsService, participantService)
	gameService := service.NewGameService(gameRepo, contestRepo, natsService)
	wsService := service.NewWebSocketService(deps.NATS, userService, participantService)
	contactService := service.NewContactService(contactRepo, deps.Config)
	inviteService := service.NewInviteService(inviteRepo, participantRepo, contestRepo, participantService, natsService)

	statsRepo := repository.NewStatsRepository(db)
	statsService := service.NewStatsService(statsRepo)

	leaderboardRepo := repository.NewLeaderboardRepository(db)
	leaderboardService := service.NewLeaderboardService(leaderboardRepo)

	contestHandler := handler.NewContestHandler(contestService)
	wsHandler := handler.NewWebSocketHandler(wsService, contestRepo, participantService, deps.Config.Server.AllowedOrigins, deps.NATS)
	contactHandler := handler.NewContactHandler(contactService)
	statsHandler := handler.NewStatsHandler(statsService)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardService)
	inviteHandler := handler.NewInviteHandler(inviteService)
	gameHandler := handler.NewGameHandler(gameService)
	participantHandler := handler.NewParticipantHandler(participantService)
	userHandler := handler.NewUserHandler(userService)
	healthHandler := handler.NewHealthHandler(db, deps.NATS, deps.OIDCVerifier)

	routes.RegisterRootRoutes(r.Group(""), healthHandler)
	routes.RegisterStatsRoutes(r.Group("/stats"), statsHandler)
	routes.RegisterLeaderboardRoutes(r.Group("/leaderboard"), leaderboardHandler, userService)
	routes.RegisterContactRoute(r.Group("/contact"), contactHandler, deps.Config.Server.ContactRateLimit)
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler, userService)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler, userService)

	routes.RegisterInviteRoutes(r.Group("/invites"), inviteHandler, userService)
	routes.RegisterContestInviteRoutes(r.Group("/contests/:id/invites"), inviteHandler, userService)

	routes.RegisterGameRoutes(r.Group("/games"), gameHandler, userService)

	routes.RegisterMyContestsRoute(r.Group("/contests/me"), participantHandler, userService)
	routes.RegisterParticipantRoutes(r.Group("/contests/:id/participants"), participantHandler, userService)

	routes.RegisterUserRoutes(r.Group("/users/me"), userHandler, userService)
}
