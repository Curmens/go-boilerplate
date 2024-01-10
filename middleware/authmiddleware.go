package middleware

import (
	"net/http"

	"example.com/utils"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware(c *gin.Context) {
	token, err := c.Cookie("token")

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
			"msg":   err,
		})
		c.Abort()
		return
	}

	if _, error := utils.ParseJwt(token); error != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
			"msg":   error,
		})
		c.Abort()
		return
	}

	c.Next()
}
