package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/routes"
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

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.PrometheusMiddleware)
	r.Use(middleware.LoggerMiddleware)

	routes.RegisterRootRoutes(r.Group(""))
	routes.RegisterSquaresRoutes(r.Group("/grids"))
	routes.RegisterWebSocketRoutes(r.Group("/ws"))

	go http.ListenAndServe(":2112", promhttp.Handler())

	r.Run(":8080")
}
