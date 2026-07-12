package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/swaggo/swag"
)

const scalarPage = `<!doctype html>
<html>
<head>
  <title>Squares API</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.62.5"></script>
  <script>
    Scalar.createApiReference('#app', { url: '/swagger/doc.json', darkMode: true })
  </script>
</body>
</html>`

func RegisterRootRoutes(rg *gin.RouterGroup, healthHandler handler.HealthHandler) {
	rg.GET("/health/live", healthHandler.Liveness)
	rg.GET("/health/ready", healthHandler.Readiness)

	rg.GET("/swagger/doc.json", serveSwaggerSpec)
	rg.GET("/swagger", serveScalar)
	rg.GET("/swagger/", serveScalar)
	rg.GET("/swagger/index.html", serveScalar)
}

func serveSwaggerSpec(c *gin.Context) {
	doc, err := swag.ReadDoc()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Data(http.StatusOK, "application/json", []byte(doc))
}

func serveScalar(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarPage))
}
