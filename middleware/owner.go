package middleware

import (
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
