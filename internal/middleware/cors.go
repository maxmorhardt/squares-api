package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		// allow localhost for development and production domain
		AllowOriginFunc: func(origin string) bool {
			if origin == "http://localhost:3000" || origin == "https://squares.maxstash.io" {
				return true
			}

			return false
		},
		// allowed http methods
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// allowed request headers
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "Cache-Control"},
		// headers exposed to client
		ExposeHeaders: []string{"Content-Length", "Content-Type"},
		// allow credentials (cookies, auth headers)
		AllowCredentials: true,
		// cache preflight requests for 12 hours
		MaxAge: 12 * time.Hour,
	})
}
