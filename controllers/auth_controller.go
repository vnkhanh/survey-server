package controllers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/middleware"
	"github.com/vnkhanh/survey-server/models"
	"github.com/vnkhanh/survey-server/services"
	"github.com/vnkhanh/survey-server/utils"
	"google.golang.org/api/idtoken"
)

func Me(c *gin.Context) {
	c.JSON(http.StatusOK, c.MustGet(middleware.CtxUserPublic))
}

type loginRequest struct {
	Email   string `json:"email" binding:"required,email"`
	MatKhau string `json:"mat_khau" binding:"required,min=6"`
}

func Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "Dữ liệu không hợp lệ",
			"error":   err.Error(),
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))

	// Tìm user theo email
	var u models.NguoiDung
	if err := config.DB.Where("email = ?", email).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Email hoặc mật khẩu không đúng"})
		return
	}

	// So khớp mật khẩu (chú ý thứ tự: hash trước, raw sau)
	if ok := utils.CheckPassword(u.MatKhau, req.MatKhau); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Email hoặc mật khẩu không đúng"})
		return
	}

	// Vai trò trong token
	role := "user"
	if u.VaiTro {
		role = "admin"
	}

	// Tạo JWT token
	token, err := utils.GenerateToken(strconv.FormatUint(uint64(u.ID), 10), role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không tạo được token"})
		return
	}

	exp := time.Now().Add(24 * time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": exp,
		"role":       role,
		"user": gin.H{
			"id":       u.ID,
			"ten":      u.Ten,
			"email":    u.Email,
			"vai_tro":  u.VaiTro,
			"ngay_tao": u.NgayTao,
		},
	})
}

type GoogleTokenRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func GoogleLoginHandler(c *gin.Context) {
	var req GoogleTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Thiếu id_token"})
		return
	}

	// Xác minh id_token với Google
	payload, err := idtoken.Validate(context.Background(), req.IDToken, services.GoogleClientID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Token Google không hợp lệ",
			"error":   err.Error(),
		})
		return
	}

	// Lấy thông tin user từ payload
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)

	// Tìm user trong DB hoặc tạo mới
	var user models.NguoiDung
	if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
		user = models.NguoiDung{
			Ten:     name,
			Email:   email,
			VaiTro:  false,
			NgayTao: time.Now(),
		}
		config.DB.Create(&user)
	}

	// Sinh JWT của hệ thống
	token, _ := utils.GenerateToken(strconv.FormatUint(uint64(user.ID), 10), "user")

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"ten":      user.Ten,
			"email":    user.Email,
			"vai_tro":  user.VaiTro,
			"ngay_tao": user.NgayTao,
		},
	})
}
