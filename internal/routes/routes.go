package routes

import (
	"context"
	"log"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	r.Use(metrics.PrometheusMiddleware)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
		})
	})

	RegisterSquaresRoutes(r)
	
	go http.ListenAndServe(":2112", promhttp.Handler())

	return r
}

func OIDCVerifier() *oidc.IDTokenVerifier {
	provider, err := oidc.NewProvider(context.Background(), "https://auth.maxstash.io/realms/maxstash")
	if err != nil {
		log.Fatal(err)
	}

	return provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
}