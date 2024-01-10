package routes

import (
	"example.com/controllers"
	"example.com/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.AuthMiddleware)
	r.GET("/ping", controllers.Ping)

	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	return r
}
