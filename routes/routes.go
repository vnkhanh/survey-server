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
		users := api.Group("/users")
		{
			users.GET("", middleware.AuthJWT(), controllers.GetUserByEmail)
		}
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
			forms.Use(middleware.AuthJWT())
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
			// Tạo link chia sẻ form
			forms.POST("/:id/share", middleware.CheckFormOwner(), controllers.ShareForm)
			// Lấy form public theo shareToken

			forms.PATCH("/:id/limit", controllers.UpdateFormLimit)    // API cập nhật giới hạn trả lời
			forms.POST("/:id/clone", controllers.CloneForm)           // Clone form (bao gồm câu hỏi + lựa chọn) // BE-32
			forms.GET("/my", controllers.GetMyForms)                  // mới thêm - Lấy form của chính user
			forms.GET("/:id/submissions", controllers.GetSubmissions) //BE-25
			forms.GET("/:id/submissions/:sub_id", controllers.GetSubmissionDetail)
			forms.GET("/:id/dashboard", controllers.GetFormDashboard)
			forms.POST("/:id/export", middleware.CheckFormEditor(), controllers.CreateExport)
		}
		api.GET("/forms/public/:shareToken", controllers.GetPublicForm) // BE-20  ĐỂ YÊN ROUTE NÀY NHA KHÔNG ĐỔI GÌ HẾT
		api.POST("/uploads", controllers.UploadFile)
		api.GET("/exports/:job_id", middleware.AuthJWT(), controllers.GetExport)

		api.PUT("/questions/:id", middleware.CheckQuestionEditor(), controllers.UpdateQuestion)    // BE-06
		api.DELETE("/questions/:id", middleware.CheckQuestionEditor(), controllers.DeleteQuestion) // BE-07
		//invites
		roomInvites := api.Group("/room-invites")
		{
			roomInvites.POST("/:id/invite", middleware.AuthJWT(), controllers.InviteUserToRoom) // gửi lời mời
			roomInvites.GET("/invites", middleware.AuthJWT(), controllers.ListRoomInvites)
			roomInvites.PUT("/:inviteID/respond", controllers.RespondToInvite) // accept / reject
			roomInvites.DELETE("/:inviteID", controllers.DeleteInvite)         // xóa lời mời
		}
		// BE-12 - 17: room
		rooms := api.Group("/rooms")
		rooms.Use(middleware.AuthJWT())
		{

			rooms.POST("", controllers.CreateRoom)                                                     //13
			rooms.GET("/:id", controllers.GetRoomDetail)                                               //14
			rooms.PUT("/:id", middleware.CheckRoomOwner(), controllers.UpdateRoom)                     //15
			rooms.DELETE("/:id", middleware.CheckRoomOwner(), controllers.DeleteRoom)                  //16
			rooms.POST("/:id/password", middleware.CheckRoomOwner(), controllers.SetRoomPassword)      //17
			rooms.DELETE("/:id/password", middleware.CheckRoomOwner(), controllers.RemoveRoomPassword) //18
			rooms.GET("/share/:shareURL", controllers.GetRoomByShareURL)
			rooms.POST("/:id/share", middleware.CheckRoomOwner(), controllers.ShareRoom) // BE-19 Tạo link chia sẻ room
			rooms.POST("/:id/enter", controllers.EnterRoom)                              // BE-22 Tham gia room

			rooms.GET("", controllers.ListRooms)
			rooms.PUT("/:id/archive", middleware.CheckRoomOwner(), controllers.ArchiveRoom)
			rooms.PUT("/:id/restore", middleware.CheckRoomOwner(), controllers.RestoreRoom)
			// API 22-2
			rooms.DELETE("/:id/removemem/:memberId", controllers.RemoveMemberFromRoom)
			// API 22-3
			rooms.GET("/:id/participants", controllers.GetRoomParticipants)                                     // BE-29
			rooms.POST("/:id/lock", middleware.CheckRoomOwner(), controllers.LockRoom)                          // BE-30
			rooms.PUT("/:id/unlock", middleware.AuthJWT(), middleware.CheckRoomOwner(), controllers.UnlockRoom) // BE-31
			rooms.GET("/lobby", controllers.GetLobbyRooms)
			rooms.GET("/archived", controllers.GetArchivedRooms)
			//BE21 Lấy danh sách room public (lobby)// Lấy danh sách room lobby (phân trang + tìm kiếm)

		}
		api.GET("/lobby", controllers.GetLobbyRooms) //BE21 Lấy danh sách room public (lobby)
		api.POST("/forms/:id/submissions", middleware.OptionalAuth(), controllers.SubmitSurvey)
		//BE-23

	}
}
