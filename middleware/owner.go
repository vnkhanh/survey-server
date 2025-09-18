package middleware

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
)

// CheckFormOwner: nạp form vào context & xác thực sở hữu (loại trừ form đã deleted)
func CheckFormOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		// user hiện tại (đã được AuthJWT set vào context với key CtxUser = "user")
		u := c.MustGet(CtxUser).(models.NguoiDung)

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "ID không hợp lệ"})
			return
		}

		var f models.KhaoSat
		// Không cho thao tác trên form đã "deleted"
		if err := config.DB.
			Where("id = ? AND trang_thai <> 'deleted'", id).
			First(&f).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Form không tồn tại"})
			return
		}

		// Chỉ owner được thao tác
		if f.NguoiTaoID == nil || *f.NguoiTaoID != u.ID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Bạn không có quyền thao tác form này"})
			return
		}

		// Đưa form vào context để controller dùng tiếp
		c.Set("formObj", f)
		c.Next()
	}
}

// CheckRoomOwner: nạp room vào context & xác thực sở hữu
func CheckRoomOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy user từ context (AuthJWT đã set)
		u, ok := c.Get(CtxUser)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Chưa đăng nhập"})
			return
		}
		user := u.(models.NguoiDung)

		// Lấy room ID từ param
		idStr := c.Param("id")
		roomID, err := strconv.Atoi(idStr)
		if err != nil || roomID <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "ID room không hợp lệ"})
			return
		}

		// Lấy room từ DB
		var room models.Room
		if err := config.DB.First(&room, roomID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Room không tồn tại"})
			return
		}

		// Debug log
		log.Printf("\033[31m[CheckRoomOwner] UserID=%d, RoomID=%d, OwnerID=%v\033[0m\n", user.ID, roomID, room.NguoiTaoID)

		// Kiểm tra quyền sở hữu
		if room.NguoiTaoID == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Room chưa có owner"})
			return
		}
		if *room.NguoiTaoID != user.ID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Bạn không có quyền thao tác room này"})
			return
		}

		// Nạp room vào context
		c.Set("roomObj", room)
		c.Next()
	}
}
