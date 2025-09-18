package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/middleware"
	"github.com/vnkhanh/survey-server/models"
)

/* ========== BE-05: Thêm câu hỏi (owner-only) ========== */

type addQuestionReq struct {
	Type    string `json:"type"    binding:"required"`
	Content string `json:"content" binding:"required"`
	Props   json.RawMessage `json:"props"`
}

func AddQuestion(c *gin.Context) {
	f := c.MustGet("formObj").(models.KhaoSat)

	var req addQuestionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Payload không hợp lệ", "error": err.Error()})
		return
	}

	// Chuẩn hoá type
	req.Type = strings.ToUpper(strings.TrimSpace(req.Type))

	// Lấy index kế tiếp = MAX(thu_tu)+1 (0-based)
	type nextRes struct{ Next int }
	var r nextRes
	_ = config.DB.Model(&models.CauHoi{}).
		Where("khao_sat_id = ?", f.ID).
		Select("COALESCE(MAX(thu_tu), -1) + 1 AS next").
		Scan(&r).Error

	if len(req.Props) > 0 && !json.Valid(req.Props) {
    c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "props không phải JSON hợp lệ"})
    return
	}	

	q := models.CauHoi{
		KhaoSatID:  f.ID,
		NoiDung:    req.Content,
		LoaiCauHoi: req.Type,
		ThuTu:      r.Next,
	}

	if len(req.Props) > 0 {
    q.PropsJSON = string(req.Props) // <-- LƯU
	}

	if err := config.DB.Create(&q).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Không thể thêm câu hỏi"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"question_id": q.ID, "form_id": f.ID})
}

/* ========== BE-06: Cập nhật câu hỏi (owner-only) ========== */

type updateQuestionReq struct {
	Content *string `json:"content"`
	Props   *json.RawMessage `json:"props"`
}

func UpdateQuestion(c *gin.Context) {
    // Middleware CheckQuestionEditor đã xác thực quyền và set sẵn question
    q := c.MustGet(middleware.CtxQuestion).(models.CauHoi)

    var req updateQuestionReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "Payload không hợp lệ", "error": err.Error()})
        return
    }

    if req.Props != nil && len(*req.Props) > 0 && !json.Valid(*req.Props) {
        c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "props không phải JSON hợp lệ"})
        return
    }

    updates := map[string]interface{}{}
    if req.Content != nil {
        updates["noi_dung"] = *req.Content
    }
    if req.Props != nil {
        updates["props_json"] = string(*req.Props)
    }
    if len(updates) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Không có gì để cập nhật"})
        return
    }

    if err := config.DB.Model(&q).Updates(updates).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Cập nhật thất bại"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

/* ========== BE-07: Xoá câu hỏi (owner-only) + dồn thứ tự ========== */

func DeleteQuestion(c *gin.Context) {
    // Middleware CheckQuestionEditor đã xác thực quyền và set sẵn question
    q := c.MustGet(middleware.CtxQuestion).(models.CauHoi)

    err := config.DB.Transaction(func(tx *gorm.DB) error {
        if err := tx.Delete(&q).Error; err != nil {
            return err
        }
        // Dồn thứ tự: các câu phía sau lùi 1 (0-based)
        if err := tx.Model(&models.CauHoi{}).
            Where("khao_sat_id = ? AND thu_tu > ?", q.KhaoSatID, q.ThuTu).
            Update("thu_tu", gorm.Expr("thu_tu - 1")).Error; err != nil {
            return err
        }
        return nil
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Xoá thất bại"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

