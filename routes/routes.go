package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/controllers"
	"github.com/vnkhanh/survey-server/middleware"
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
			auth.POST("/login", controllers.Login)
			auth.POST("/google/login", controllers.GoogleLoginHandler)
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
		forms := api.Group("/forms")
		forms.Use(middleware.AuthJWT())
		{
			// BE-01..04
			forms.POST("", controllers.CreateForm)
			forms.GET("/:id", controllers.GetFormDetail)
			forms.PUT("/:id", middleware.CheckFormOwner(), controllers.UpdateForm)
			forms.DELETE("/:id", middleware.CheckFormOwner(), controllers.DeleteForm)

			// (tuỳ chọn) Archive/Restore
			forms.PUT("/:id/archive", middleware.CheckFormOwner(), controllers.ArchiveForm)
			forms.PUT("/:id/restore", middleware.CheckFormOwner(), controllers.RestoreForm)

			// BE-05: thêm câu hỏi
			forms.POST("/:id/questions", middleware.CheckFormOwner(), controllers.AddQuestion)

			// BE-08: reorder
			forms.PUT("/:id/questions/reorder", middleware.CheckFormOwner(), controllers.ReorderQuestions)

			// BE-09..10: settings
			forms.PUT("/:id/settings", middleware.CheckFormOwner(), controllers.UpdateFormSettings)
			forms.GET("/:id/settings", controllers.GetFormSettings)
		}

		api.PUT("/questions/:id", middleware.AuthJWT(), controllers.UpdateQuestion)    // BE-06
    	api.DELETE("/questions/:id", middleware.AuthJWT(), controllers.DeleteQuestion) // BE-07

	}
}
