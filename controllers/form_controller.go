package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/vnkhanh/survey-server/utils"
	"github.com/vnkhanh/survey-server/middleware"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
)

/* ========== BE-01: Tạo biểu mẫu khảo sát ========== */

type createFormReq struct {
	Title       string          `json:"title"       binding:"required,min=1"`
	Description string          `json:"description"`
	TemplateID  *uint           `json:"template_id"` // để mở rộng
	Settings    json.RawMessage `json:"settings"`    // JSON tuỳ chọn (lưu TEXT)
	Theme       json.RawMessage `json:"theme"`       // JSON tuỳ chọn (lưu TEXT)
}

func CreateForm(c *gin.Context) {

	var req createFormReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Payload không hợp lệ", "error": err.Error()})
		return
	}

	// Validate JSON fields
	if len(req.Settings) > 0 && !json.Valid(req.Settings) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "settings không phải JSON hợp lệ"})
		return
	}
	if len(req.Theme) > 0 && !json.Valid(req.Theme) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "theme không phải JSON hợp lệ"})
		return
	}

	var ownerID *uint
	if v, ok := c.Get(middleware.CtxUser); ok {
		if u, ok2 := v.(models.NguoiDung); ok2 {
			ownerID = &u.ID
		}
	}
	
	form := models.KhaoSat{
		TieuDe:     req.Title,
		MoTa:       req.Description,
		NguoiTaoID: ownerID,
		TrangThai:  "active",
		TemplateID: req.TemplateID,
	}
	if len(req.Settings) > 0 {
		form.SettingsJSON = string(req.Settings)
	}
	if len(req.Theme) > 0 {
		form.ThemeJSON = string(req.Theme)
	}

	var rawToken string
	var err error
	if ownerID == nil {
		rawToken, err = utils.GenerateEditToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể sinh edit token"})
			return
		}
		if form.EditTokenHash, err = utils.HashEditToken(rawToken); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể băm edit token"})
			return
		}
	}

	if err := config.DB.Create(&form).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể tạo form"})
		return
	}

	resp := gin.H{
		"id":          form.ID,
		"title":       form.TieuDe,
		"description": form.MoTa,
		"owner_id":    form.NguoiTaoID,
		"created_at":  form.NgayTao,
	}
	if ownerID == nil && rawToken != "" {
		resp["edit_token"] = rawToken
	}
	c.JSON(http.StatusCreated, resp)
}

/* ========== BE-02: Xem chi tiết form ========== */

type QuestionDTO struct {
    ID      uint             `json:"id"`
    Type    string           `json:"type"`
    Content string           `json:"content"`
    Order   int              `json:"order"`
    Props   interface{}      `json:"props,omitempty"`
    Options []models.LuaChon `json:"options,omitempty"`
}

func GetFormDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID không hợp lệ"})
		return
	}

	var form models.KhaoSat
	err = config.DB.
		Where("id = ? AND trang_thai <> 'deleted'", id).
		Preload("CauHois", func(db *gorm.DB) *gorm.DB { return db.Order("thu_tu ASC, id ASC") }).
		Preload("CauHois.LuaChons", func(db *gorm.DB) *gorm.DB { return db.Order("thu_tu ASC, id ASC") }).
		First(&form).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Form không tồn tại"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể lấy form"})
		return
	}

	var settings, theme interface{}
	if form.SettingsJSON != "" {
		_ = json.Unmarshal([]byte(form.SettingsJSON), &settings)
	}
	if form.ThemeJSON != "" {
		_ = json.Unmarshal([]byte(form.ThemeJSON), &theme)
	}

	out := make([]QuestionDTO, 0, len(form.CauHois))
	for _, q := range form.CauHois {
    	var props interface{}
    	if q.PropsJSON != "" {
        	_ = json.Unmarshal([]byte(q.PropsJSON), &props)
    	}
    	out = append(out, QuestionDTO{
        	ID: q.ID, Type: q.LoaiCauHoi, Content: q.NoiDung, Order: q.ThuTu,
        	Props: props, Options: q.LuaChons,
    	})
	}
	c.JSON(http.StatusOK, gin.H{
		"id":          form.ID,
		"title":       form.TieuDe,
		"description": form.MoTa,
		"settings":    settings,
		"theme":       theme,
		"questions":   out,
	})
}

/* ========== BE-03: Cập nhật form (owner-only) ========== */

type updateFormReq struct {
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	Settings    *json.RawMessage `json:"settings"`
}

func UpdateForm(c *gin.Context) {
	f := c.MustGet("formObj").(models.KhaoSat)

	var req updateFormReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Payload không hợp lệ", "error": err.Error()})
		return
	}

	if req.Settings != nil && len(*req.Settings) > 0 && !json.Valid(*req.Settings) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "settings không phải JSON hợp lệ"})
		return
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["tieu_de"] = *req.Title
	}
	if req.Description != nil {
		updates["mo_ta"] = *req.Description
	}
	if req.Settings != nil {
		updates["settings_json"] = string(*req.Settings)
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Không có gì để cập nhật"})
		return
	}

	if err := config.DB.Model(&models.KhaoSat{}).
		Where("id = ?", f.ID).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cập nhật thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

/* ========== BE-04: Xoá form (owner-only) — xóa mềm ========== */

func DeleteForm(c *gin.Context) {
	f := c.MustGet("formObj").(models.KhaoSat)
	if err := config.DB.Model(&models.KhaoSat{}).
		Where("id = ?", f.ID).
		Update("trang_thai", "deleted").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Xoá (mềm) thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

/* ========== BE-04 (tuỳ chọn): Archive/Restore bằng trang_thai ========== */

func ArchiveForm(c *gin.Context) {
	f := c.MustGet("formObj").(models.KhaoSat)
	if err := config.DB.Model(&models.KhaoSat{}).
		Where("id = ?", f.ID).
		Update("trang_thai", "archived").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Archive thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "archived"})
}

func RestoreForm(c *gin.Context) {
	f := c.MustGet("formObj").(models.KhaoSat)
	if err := config.DB.Model(&models.KhaoSat{}).
		Where("id = ?", f.ID).
		Update("trang_thai", "active").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Restore thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "restored"})
}

/* ========== BE-08: Sắp xếp lại câu hỏi (owner-only) ========== */

type reorderReq struct {
	Order []uint `json:"order" binding:"required,min=1,dive,required"`
}

func ReorderQuestions(c *gin.Context) {
	f := c.MustGet("formObj").(models.KhaoSat)

	var req reorderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Payload không hợp lệ", "error": err.Error()})
		return
	}

	// Validate: tất cả qID đều thuộc form
	var count int64
	if err := config.DB.Model(&models.CauHoi{}).
		Where("khao_sat_id = ? AND id IN ?", f.ID, req.Order).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể validate câu hỏi"})
		return
	}
	if count != int64(len(req.Order)) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Danh sách order chứa câu hỏi không thuộc form"})
		return
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		for idx, qID := range req.Order {
			if err := tx.Model(&models.CauHoi{}).
				Where("id = ? AND khao_sat_id = ?", qID, f.ID).
				Update("thu_tu", idx).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Cập nhật thứ tự thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

/* ========== BE-09: Cập nhật cài đặt form (owner-only) ========== */

type updateSettingsReq struct {
	Settings json.RawMessage `json:"settings" binding:"required"`
}

func UpdateFormSettings(c *gin.Context) {
	f := c.MustGet("formObj").(models.KhaoSat)

	var req updateSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Payload không hợp lệ", "error": err.Error()})
		return
	}
	if len(req.Settings) == 0 || !json.Valid(req.Settings) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "settings không phải JSON hợp lệ"})
		return
	}
	if err := config.DB.Model(&models.KhaoSat{}).
		Where("id = ?", f.ID).
		Update("settings_json", string(req.Settings)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Lưu settings thất bại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

/* ========== BE-10: Lấy cài đặt form ========== */

func GetFormSettings(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID không hợp lệ"})
		return
	}

	var f models.KhaoSat
	if e := config.DB.Select("id, settings_json").
		Where("id = ? AND trang_thai <> 'deleted'", id).
		First(&f).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Form không tồn tại"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể lấy settings"})
		return
	}

	var settings interface{}
	if f.SettingsJSON != "" {
		_ = json.Unmarshal([]byte(f.SettingsJSON), &settings)
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings})
}
