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
		{
			forms.POST("", middleware.RateLimitFormsCreate(), controllers.CreateForm) // BE-01
			forms.GET("/:id", controllers.GetFormDetail)                              // BE-02
			forms.GET("/:id/settings", controllers.GetFormSettings)                   // BE-10
			// Ghi: cần quyền editor (JWT owner hoặc Edit Token)
			forms.PUT("/:id", middleware.CheckFormEditor(), controllers.UpdateForm)                         // BE-03
			forms.DELETE("/:id", middleware.CheckFormEditor(), controllers.DeleteForm)                      // BE-04
			forms.PUT("/:id/archive", middleware.CheckFormEditor(), controllers.ArchiveForm)                // BE-04
			forms.PUT("/:id/restore", middleware.CheckFormEditor(), controllers.RestoreForm)                // BE-04
			forms.POST("/:id/questions", middleware.CheckFormEditor(), controllers.AddQuestion)             // BE-05
			forms.PUT("/:id/questions/reorder", middleware.CheckFormEditor(), controllers.ReorderQuestions) // BE-08
			forms.PUT("/:id/settings", middleware.CheckFormEditor(), controllers.UpdateFormSettings)        // BE-09
		}

		api.PUT("/questions/:id", middleware.CheckQuestionEditor(), controllers.UpdateQuestion)    // BE-06
		api.DELETE("/questions/:id", middleware.CheckQuestionEditor(), controllers.DeleteQuestion) // BE-07

		// BE-12 - 17: room
		rooms := api.Group("/rooms")
		rooms.Use(middleware.AuthJWT())
		{
			rooms.POST("", controllers.CreateRoom)
			rooms.GET("/:id", controllers.GetRoomDetail)
			rooms.GET("", controllers.ListRooms)
			rooms.PUT("/:id", middleware.CheckRoomOwner(), controllers.UpdateRoom)
			rooms.DELETE("/:id", middleware.CheckRoomOwner(), controllers.DeleteRoom)
			rooms.PUT("/:id/archive", middleware.CheckRoomOwner(), controllers.ArchiveRoom)
			rooms.PUT("/:id/restore", middleware.CheckRoomOwner(), controllers.RestoreRoom)
			rooms.POST("/:id/set-password", middleware.CheckRoomOwner(), controllers.SetRoomPassword)
		}
	}
}
