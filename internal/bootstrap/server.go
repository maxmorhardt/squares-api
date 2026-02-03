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
	config.InitRedis()

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
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware)

	if config.Env().Server.MetricsEnabled {
		r.Use(middleware.PrometheusMiddleware)
	}
}

func setupRoutes(r *gin.Engine) {
	contestRepo := repository.NewContestRepository()
	contactRepo := repository.NewContactRepository()

	authService := service.NewAuthService()
	redisService := service.NewRedisService()
	contestService := service.NewContestService(contestRepo, redisService, authService)
	wsService := service.NewWebSocketService()
	contactService := service.NewContactService(contactRepo)

	contestHandler := handler.NewContestHandler(contestService, authService)
	wsHandler := handler.NewWebSocketHandler(wsService, contestRepo)
	contactHandler := handler.NewContactHandler(contactService)

	routes.RegisterRootRoutes(r.Group(""))
	routes.RegisterContactRoute(r.Group("/contact"), contactHandler)
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler)
}