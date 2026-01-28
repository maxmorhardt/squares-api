package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/routes"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/validators"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(logger)

	logger.Info("initialized logger")
}

// @title           Squares API
// @version         1.0.0
// @description     API for squares.maxstash.io
// @securityDefinitions.apikey BearerAuth
// @type apiKey
// @in header
// @name Authorization
func main() {
	_ = godotenv.Load()

	config.InitOIDC(false)
	config.InitDB()
	config.InitSMTP()

	go config.InitRedis()
	
	if err := initGin().Run(":8080"); err != nil {
		slog.Error("failed to start server", "error", err)
		panic(err)
	}
}

func initGin() *gin.Engine {
	r := gin.New()

	metricsEnabled := os.Getenv("METRICS_ENABLED") == "true"

	setupMiddleware(r, metricsEnabled)
	setupRoutes(r)
	setupMetricsServer(metricsEnabled)
	setupValidators()

	return r
}

func setupMetricsServer(metricsEnabled bool) {
	if !metricsEnabled {
		slog.Info("metrics disabled")
		return
	}

	slog.Info("starting metrics server on :9090")
	go func() {
		if err := http.ListenAndServe(":9090", promhttp.Handler()); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()
}

func setupMiddleware(r *gin.Engine, metricsEnabled bool) {
	r.Use(gin.Recovery())
	r.Use(middleware.RequestSizeLimitMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.LoggerMiddleware)

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

func setupValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("contestname", validators.ValidateContestName)
	}
}
