package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
	"github.com/vnkhanh/survey-server/utils"
)

// AuthJWT kiểm tra Authorization: Bearer <token>, validate JWT, lấy user và inject vào context.
func AuthJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Missing or invalid Authorization header"})
			return
		}
		rawToken := strings.TrimSpace(authHeader[7:])

		claims, err := utils.VerifyToken(rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			return
		}

		// UserID trong claims là string → parse ra uint64 để tìm DB theo primary key
		uid, err := strconv.ParseUint(claims.UserID, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid subject"})
			return
		}

		var user models.NguoiDung
		if err := config.DB.First(&user, uid).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "User not found"})
			return
		}

		// Inject vào context
		c.Set(CtxUser, user)
        c.Set(CtxUserPublic, gin.H{
			"id":       user.ID,
			"ten":      user.Ten,
			"email":    user.Email,
			"vai_tro":  user.VaiTro,
			"ngay_tao": user.NgayTao,
		})

		c.Next()
	}
}

// RequireAdmin chặn các route chỉ dành cho admin
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}
		u := v.(models.NguoiDung)
		if !u.VaiTro {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Forbidden"})
			return
		}
		c.Next()
	}
}
