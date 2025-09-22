package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SubmitFormAnswer(c *gin.Context) {
	formID := c.Param("id")

	var req struct {
		CauTraLoi string `json:"cau_tra_loi"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Payload không hợp lệ"})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var form models.KhaoSat

		// Lấy form với row lock
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&form, "id = ?", formID).Error; err != nil {
			fmt.Println("Lỗi khi lấy form:", err)
			c.JSON(http.StatusNotFound, gin.H{"message": "Form không tồn tại"})
			return err
		}

		// Kiểm tra giới hạn trả lời
		if form.GioiHanTL != nil && form.SoLanTraLoi >= *form.GioiHanTL {
			c.JSON(http.StatusForbidden, gin.H{"message": "Đã đạt giới hạn số lần trả lời"})
			return gorm.ErrInvalidTransaction
		}

		// Tăng số lần trả lời
		form.SoLanTraLoi++
		if err := tx.Save(&form).Error; err != nil {
			fmt.Println("Lỗi khi cập nhật số lần trả lời:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể cập nhật số lần trả lời"})
			return err
		}

		// Lưu câu trả lời
		answer := models.Answer{
			KhaoSatID: form.ID,
			CauTraLoi: req.CauTraLoi,
		}
		if err := tx.Create(&answer).Error; err != nil {
			fmt.Println("Lỗi khi lưu câu trả lời:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể lưu câu trả lời"})
			return err
		}

		return nil
	})

	if err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Gửi câu trả lời thành công"})
	}
}
