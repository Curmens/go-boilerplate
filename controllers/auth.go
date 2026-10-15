package controllers

import (
	"example.com/middleware"
	"example.com/utils"

	"github.com/gin-gonic/gin"
)

// Me returns the authenticated caller's token claims — an example protected handler
func Me(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		utils.Unauthorized(c, "")
		return
	}

	utils.Success(c, gin.H{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"role":    claims.Role,
	})
}
