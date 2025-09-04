package controllers

import (
    "net/http"
	"strconv"
	"strings"
	"time"

    "github.com/gin-gonic/gin"
    "github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/middleware"
    "github.com/vnkhanh/survey-server/models"
    "github.com/vnkhanh/survey-server/utils"
)

type DangKyReq struct {
    Ten     string `json:"ten" binding:"required,min=1"`
    Email   string `json:"email" binding:"required,email"`
    MatKhau string `json:"mat_khau" binding:"required,min=6"`
}

func Register(c *gin.Context) {
    var req DangKyReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusUnprocessableEntity, gin.H{"message": err.Error()})
        return
    }

	email := strings.TrimSpace(strings.ToLower(req.Email))

    var count int64
    config.DB.Model(&models.NguoiDung{}).Where("email = ?", email).Count(&count)
    if count > 0 {
        c.JSON(http.StatusConflict, gin.H{"message": "Email đã tồn tại"})
        return
    }

    hash, err := utils.HashPassword(req.MatKhau)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể mã hóa mật khẩu"})
        return
    }

    nd := models.NguoiDung{
        Ten:     req.Ten,
        Email:   email,
        MatKhau: hash,
        VaiTro:  false,
    }

    if err := config.DB.Create(&nd).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể tạo tài khoản"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "user": gin.H{
            "id":       nd.ID,
            "ten":      nd.Ten,
            "email":    nd.Email,
            "vai_tro":  nd.VaiTro,
            "ngay_tao": nd.NgayTao,
        },
    })
}

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
		"token": token,
		"expires_at": exp,
        "role": role,
		"user": gin.H{
			"id":       u.ID,
			"ten":      u.Ten,
			"email":    u.Email,
			"vai_tro":  u.VaiTro,
			"ngay_tao": u.NgayTao,
		},
	})
}