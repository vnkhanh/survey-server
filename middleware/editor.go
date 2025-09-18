package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
	"github.com/vnkhanh/survey-server/utils"
	"gorm.io/gorm"
)

const (
	HeaderEditToken = "X-Form-Edit-Token"
	CtxForm         = "formObj"     // form đã nạp sẵn
	CtxQuestion     = "questionObj" // question đã nạp sẵn
)

// helper: kiểm tra owner an toàn với con trỏ *uint
func isOwner(u models.NguoiDung, f *models.KhaoSat) bool {
	return f.NguoiTaoID != nil && *f.NguoiTaoID == u.ID
}

// CheckFormEditor: cho phép nếu (1) có JWT và là owner, hoặc (2) có edit token hợp lệ.
func CheckFormEditor() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "ID không hợp lệ"})
			return
		}

		var f models.KhaoSat
		if e := config.DB.Where("id = ? AND trang_thai <> 'deleted'", id).First(&f).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Form không tồn tại"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Không thể đọc form"})
			return
		}

		// 1) Nếu có JWT & là owner
		if v, ok := c.Get(CtxUser); ok {
			if u, ok2 := v.(models.NguoiDung); ok2 && isOwner(u, &f) {
				c.Set(CtxForm, f)
				c.Next()
				return
			}
		}

		// 2) Kiểm tra edit token
		token := c.GetHeader(HeaderEditToken)
		if token != "" && utils.VerifyEditToken(f.EditTokenHash, token) {
			c.Set(CtxForm, f)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Thiếu hoặc sai quyền chỉnh sửa form"})
	}
}

// CheckQuestionEditor: giống CheckFormEditor nhưng tra ngược từ question -> form
func CheckQuestionEditor() gin.HandlerFunc {
	return func(c *gin.Context) {
		qid, err := strconv.Atoi(c.Param("id"))
		if err != nil || qid <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "ID không hợp lệ"})
			return
		}

		var q models.CauHoi
		if e := config.DB.Select("id, khao_sat_id, thu_tu").
			First(&q, qid).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Câu hỏi không tồn tại"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Không thể đọc câu hỏi"})
			return
		}

		var f models.KhaoSat
		if e := config.DB.Select("id, nguoi_tao_id, trang_thai, edit_token_hash").
			Where("id = ? AND trang_thai <> 'deleted'", q.KhaoSatID).
			First(&f).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"message": "Form không tồn tại hoặc đã xoá"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "Không thể đọc form"})
			return
		}

		// 1) JWT owner
		if v, ok := c.Get(CtxUser); ok {
			if u, ok2 := v.(models.NguoiDung); ok2 && isOwner(u, &f) {
				c.Set(CtxQuestion, q)
				c.Next()
				return
			}
		}

		// 2) Edit token
		token := c.GetHeader(HeaderEditToken)
		if token != "" && utils.VerifyEditToken(f.EditTokenHash, token) {
			c.Set(CtxQuestion, q)
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "Thiếu hoặc sai quyền chỉnh sửa câu hỏi"})
	}
}
