package controllers

import (
	"fmt"
	"net/http"

	"example.com/docs"

	"github.com/gin-gonic/gin"
)

const swaggerUIVersion = "5.17.14"

// OpenAPISpecHandler serves the raw OpenAPI YAML spec embedded at compile time
func OpenAPISpecHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", docs.OpenAPISpec)
	}
}

// SwaggerUIHandler serves a Swagger UI page pointed at the local spec endpoint
func SwaggerUIHandler(title, specURL string) gin.HandlerFunc {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%[1]s – API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@%[3]s/swagger-ui.css" />
  <style>body { margin: 0; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@%[3]s/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({ url: %[2]q, dom_id: '#swagger-ui', deepLinking: true })
    }
  </script>
</body>
</html>`, title, specURL, swaggerUIVersion)

	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}
