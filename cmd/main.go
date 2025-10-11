package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/db"
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
	db.Init()

	env := os.Getenv("APP_ENV")

	switch env {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	default:
		gin.SetMode(gin.DebugMode)
	}
	
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.PrometheusMiddleware)
	r.Use(middleware.LoggerMiddleware)

	routes.RegisterRootRoutes(r.Group("/"))
	routes.RegisterSquaresRoutes(r.Group("/grids"))
	routes.RegisterSSERoutes(r.Group("/events"))
	
	go http.ListenAndServe(":2112", promhttp.Handler())

	r.Run(":8080")
}