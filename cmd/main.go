package main

import (
	"log/slog"
	"net/http"
	"os"

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
	config.InitJWT()
	config.InitSMTP()

	initGin().Run(":8080")
}

func initGin() *gin.Engine {
	r := gin.New()

	metricsEnabled := os.Getenv("METRICS_ENABLED") == "true"

	setupMiddleware(r, metricsEnabled)
	setupRoutes(r)
	setupMetricsServer(metricsEnabled)

	return r
}

func setupMetricsServer(metricsEnabled bool) {
	if !metricsEnabled {
		slog.Info("metrics disabled")
		return
	}

	slog.Info("starting metrics server on :2112")
	go func() {
		if err := http.ListenAndServe(":2112", promhttp.Handler()); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()
}

func setupMiddleware(r *gin.Engine, metricsEnabled bool) {
	r.Use(gin.Recovery())
	r.Use(middleware.RequestSizeLimitMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware)
	r.Use(middleware.RateLimitMiddleware())
	
	if metricsEnabled {
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

	routes.RegisterRootRoutes(r.Group(""), contactHandler)
	routes.RegisterContestRoutes(r.Group("/contests"), contestHandler)
	routes.RegisterWebSocketRoutes(r.Group("/ws"), wsHandler)
}
