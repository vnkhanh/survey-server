package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/controllers"
)

func SetupRoutes(r *gin.Engine) {
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/health", controllers.HealthCheck)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", controllers.Register)
		}
		protected := api.Group("/")
		protected.Use(middleware.AuthJWT())
		{
			protected.GET("/me", controllers.Me)
		}

		admin := protected.Group("/admin")
		admin.Use(middleware.RequireAdmin())
		{
			admin.GET("/only", func(c *gin.Context) {
				c.JSON(200, gin.H{"ok": true})
			})
		}
	}
}
