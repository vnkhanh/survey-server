package controllers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/vnkhanh/survey-server/config"
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

    var count int64
    config.DB.Model(&models.NguoiDung{}).Where("email = ?", req.Email).Count(&count)
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
        Email:   req.Email,
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