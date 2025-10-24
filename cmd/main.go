package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/routes"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// @title           Squares API
// @version         1.0.0
// @description     API for squares.maxstash.io
// @securityDefinitions.apikey BearerAuth
// @type apiKey
// @in header
// @name Authorization
func main() {
	godotenv.Load()

	config.InitDB()
	config.InitRedis()

	initGin().Run(":8080")
}

func initGin() *gin.Engine {
	r := gin.New()

	setupMiddleware(r)
	setupRoutes(r)

	go http.ListenAndServe(":2112", promhttp.Handler())

	return r
}

func setupMiddleware(r *gin.Engine) {
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.PrometheusMiddleware)
	r.Use(middleware.LoggerMiddleware)
	r.Use(middleware.RateLimitMiddleware())
}

func setupRoutes(r *gin.Engine) {
	contestRepo := repository.NewContestRepository()

	authService := service.NewAuthService()
	redisService := service.NewRedisService()
	validationService := service.NewValidationService(contestRepo)

	contestService := service.NewContestService(contestRepo, redisService, authService)
	contestHandler := handler.NewContestHandler(contestService, authService, validationService)

	wsService := service.NewWebSocketService()
	wsHandler := handler.NewWebSocketHandler(wsService, validationService)

	routes.RegisterRootRoutes(r.Group(""))
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler)
}
