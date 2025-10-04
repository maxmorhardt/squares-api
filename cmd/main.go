package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/db"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/routes"
	"github.com/maxmorhardt/squares-api/pkg/logger"
	"github.com/maxmorhardt/squares-api/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Squares API
// @version         1.0
// @description     API for squares.maxstash.io

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	godotenv.Load()
	db.Init()
	
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(metrics.PrometheusMiddleware)
	r.Use(logger.LoggerMiddleware)

	r.GET("/health", handler.HealthCheck)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	routes.RegisterSquaresRoutes(r.Group("/grid"))
	
	go http.ListenAndServe(":2112", promhttp.Handler())

	r.Run(":8080")
}