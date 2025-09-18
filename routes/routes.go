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

			// BE-11: themes form
			forms.PUT("/:id/theme", middleware.CheckFormOwner(), controllers.UpdateFormTheme)
			forms.GET("/:id/theme", controllers.GetFormTheme)
		}

		api.PUT("/questions/:id", middleware.AuthJWT(), controllers.UpdateQuestion)    // BE-06
		api.DELETE("/questions/:id", middleware.AuthJWT(), controllers.DeleteQuestion) // BE-07

		// BE-12: room
		rooms := api.Group("/rooms")
		rooms.Use(middleware.AuthJWT())
		{
			rooms.POST("", controllers.CreateRoom)                                                     //13
			rooms.GET("/:id", controllers.GetRoomDetail)                                               //14
			rooms.PUT("/:id", middleware.CheckRoomOwner(), controllers.UpdateRoom)                     //15
			rooms.DELETE("/:id", middleware.CheckRoomOwner(), controllers.DeleteRoom)                  //16
			rooms.POST("/:id/password", middleware.CheckRoomOwner(), controllers.SetRoomPassword)      //17
			rooms.DELETE("/:id/password", middleware.CheckRoomOwner(), controllers.RemoveRoomPassword) //18
			rooms.POST("/:id/share", middleware.CheckRoomOwner(), controllers.CreateRoomShare)         // 19
			forms.POST("/:id/share", middleware.CheckFormOwner(), controllers.CreateFormShare)         //20
			rooms.POST("/:id/enter", controllers.EnterRoom)                                            // BE-22 Tham gia room
		}
		api.GET("/lobby", controllers.GetLobbyRooms) //BE21 Lấy danh sách room public (lobby)
	}
}
