package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
	"gorm.io/gorm"
)

type AnswerReq struct {
	CauHoiID   uint   `json:"cau_hoi_id" binding:"required"`
	LoaiCauHoi string `json:"loai_cau_hoi" binding:"required"`
	NoiDung    string `json:"noi_dung"` // cho fill_blank, rating, true_false, upload_file
	LuaChon    string `json:"lua_chon"` // cho multiple_choice (JSON array string)
}

type SubmitSurveyReq struct {
	KhaoSatID uint        `json:"khao_sat_id" binding:"required"`
	Email     *string     `json:"email"` // cho khách nhập
	Answers   []AnswerReq `json:"answers" binding:"required"`
}

func SubmitSurvey(c *gin.Context) {
	// Lấy id khảo sát từ param
	surveyIDStr := c.Param("id")
	surveyID, err := strconv.Atoi(surveyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID khảo sát không hợp lệ"})
		return
	}

	// Lấy khảo sát từ DB
	var ks models.KhaoSat
	if err := config.DB.First(&ks, surveyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Khảo sát không tồn tại"})
		return
	}

	// Parse settings_json
	type surveySettings struct {
		RequireLogin bool `json:"require_login"` // <-- thêm flag rõ ràng
		CollectEmail bool `json:"collect_email"`
		MaxResponses *int `json:"max_responses"`
	}
	var settings surveySettings
	_ = json.Unmarshal([]byte(ks.SettingsJSON), &settings)

	// Parse body
	var req SubmitSurveyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu gửi không hợp lệ"})
		return
	}

	// Lấy user nếu có login
	var userID *uint
	if u, exists := c.Get("user"); exists {
		if user, ok := u.(models.NguoiDung); ok {
			userID = &user.ID
		}
	}

	// Nếu khảo sát bắt buộc login mà không có user
	if settings.RequireLogin && userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Khảo sát này yêu cầu đăng nhập"})
		return
	}

	tx := config.DB.Begin()

	// Xác định số lần gửi
	lanGui := 1
	if userID != nil {
		var lastPhanHoi models.PhanHoi
		if err := tx.Where("khao_sat_id = ? AND nguoi_dung_id = ?", surveyID, *userID).
			Order("lan_gui DESC").First(&lastPhanHoi).Error; err == nil {
			lanGui = lastPhanHoi.LanGui + 1
		}
	}

	// Email: nếu bắt buộc thì dùng email user, còn không thì lấy từ req
	var emailPtr *string
	if userID != nil {
		// user login → lưu email user
		var user models.NguoiDung
		_ = config.DB.First(&user, *userID)
		if user.Email != "" {
			emailPtr = &user.Email
		}
	} else if req.Email != nil && *req.Email != "" {
		// khách nhập email
		emailPtr = req.Email
	}

	submission := models.PhanHoi{
		KhaoSatID:   uint(surveyID),
		NguoiDungID: userID,   // null nếu khách
		Email:       emailPtr, // null nếu không có
		NgayGui:     time.Now(),
		LanGui:      lanGui,
	}

	if err := tx.Create(&submission).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu phản hồi"})
		return
	}

	// Lưu câu trả lời
	for _, ans := range req.Answers {
		ct := models.CauTraLoi{
			PhanHoiID: submission.ID,
			CauHoiID:  ans.CauHoiID,
		}
		if ans.LoaiCauHoi == "multiple_choice" {
			ct.LuaChon = ans.LuaChon
		} else {
			ct.NoiDung = ans.NoiDung
		}
		if err := tx.Create(&ct).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu câu trả lời"})
			return
		}
	}

	// Tăng số phản hồi
	if err := tx.Model(&models.KhaoSat{}).
		Where("id = ?", surveyID).
		UpdateColumn("so_phan_hoi", gorm.Expr("so_phan_hoi + ?", 1)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật số phản hồi"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message":  "Gửi khảo sát thành công",
		"phan_hoi": submission,
	})
}
