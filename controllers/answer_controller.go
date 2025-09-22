package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
	"github.com/vnkhanh/survey-server/utils"
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
	// 1. Lấy id khảo sát
	surveyID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID khảo sát không hợp lệ"})
		return
	}

	// 2. Lấy khảo sát
	var ks models.KhaoSat
	if err := config.DB.First(&ks, surveyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Khảo sát không tồn tại"})
		return
	}

	// 3. Parse settings_json
	var settings struct {
		RequireLogin bool `json:"require_login"`
		CollectEmail bool `json:"collect_email"`
		MaxResponses *int `json:"max_responses"`
	}
	if err := json.Unmarshal([]byte(ks.SettingsJSON), &settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cấu hình khảo sát không hợp lệ"})
		return
	}

	// 4. Kiểm tra giới hạn số phản hồi
	if settings.MaxResponses != nil {
		var count int64
		if err := config.DB.Model(&models.PhanHoi{}).Where("khao_sat_id = ?", surveyID).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra số phản hồi"})
			return
		}
		if count >= int64(*settings.MaxResponses) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Khảo sát đã đạt giới hạn số phản hồi"})
			return
		}
	}

	// 5. Parse request body - QUAN TRỌNG: Xử lý multipart form
	var req SubmitSurveyReq

	// Kiểm tra xem request có phải là multipart form không
	if strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		// Xử lý multipart form
		data := c.PostForm("data")
		if data == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu dữ liệu form"})
			return
		}

		if err := json.Unmarshal([]byte(data), &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu JSON không hợp lệ"})
			return
		}
	} else {
		// Xử lý JSON request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu gửi không hợp lệ: " + err.Error()})
			return
		}
	}

	// 6. Validate email nếu có
	if req.Email != nil && *req.Email != "" {
		if !isValidEmail(*req.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email không hợp lệ"})
			return
		}
	}

	// 7. Kiểm tra đăng nhập
	var userID *uint
	if u, exists := c.Get("user"); exists {
		if user, ok := u.(models.NguoiDung); ok {
			userID = &user.ID
		}
	}
	if settings.RequireLogin && userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Khảo sát này yêu cầu đăng nhập"})
		return
	}

	// 8. Validate câu trả lời
	for _, ans := range req.Answers {
		var q models.CauHoi
		if err := config.DB.
			Where("id = ? AND khao_sat_id = ?", ans.CauHoiID, surveyID).
			First(&q).Error; err != nil {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": fmt.Sprintf("Câu hỏi %d không hợp lệ", ans.CauHoiID)})
			return
		}

		var props struct {
			Required bool `json:"required"`
		}
		if q.PropsJSON != "" {
			if err := json.Unmarshal([]byte(q.PropsJSON), &props); err != nil {
				log.Printf("Lỗi parse props JSON cho câu hỏi %d: %v", ans.CauHoiID, err)
			}
		}

		if props.Required {
			switch strings.ToUpper(q.LoaiCauHoi) {
			case "MULTIPLE_CHOICE":
				if strings.TrimSpace(ans.LuaChon) == "" || ans.LuaChon == "[]" {
					c.JSON(http.StatusBadRequest,
						gin.H{"error": fmt.Sprintf("Câu hỏi %d là bắt buộc", ans.CauHoiID)})
					return
				}
			case "UPLOAD_FILE":
				// Kiểm tra file trong form data
				fileKey := fmt.Sprintf("file_%d", ans.CauHoiID)
				if _, err := c.FormFile(fileKey); err != nil {
					c.JSON(http.StatusBadRequest,
						gin.H{"error": fmt.Sprintf("Thiếu file cho câu hỏi %d", ans.CauHoiID)})
					return
				}
			default:
				if strings.TrimSpace(ans.NoiDung) == "" {
					c.JSON(http.StatusBadRequest,
						gin.H{"error": fmt.Sprintf("Câu hỏi %d là bắt buộc", ans.CauHoiID)})
					return
				}
			}
		}
	}

	// 9. Chuẩn bị phản hồi
	emailPtr := req.Email
	if userID != nil {
		var user models.NguoiDung
		if err := config.DB.First(&user, *userID).Error; err == nil && user.Email != "" {
			emailPtr = &user.Email
		}
	}

	lanGui := 1
	if userID != nil {
		var last models.PhanHoi
		if err := config.DB.Where("khao_sat_id = ? AND nguoi_dung_id = ?", surveyID, *userID).
			Order("lan_gui DESC").First(&last).Error; err == nil {
			lanGui = last.LanGui + 1
		}
	}

	// === Transaction đảm bảo rollback nếu lỗi ===
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		submission := models.PhanHoi{
			KhaoSatID:   uint(surveyID),
			NguoiDungID: userID,
			Email:       emailPtr,
			NgayGui:     time.Now(),
			LanGui:      lanGui,
		}
		if err := tx.Create(&submission).Error; err != nil {
			return err
		}

		// 10. Lưu từng câu trả lời
		for _, ans := range req.Answers {
			var q models.CauHoi
			if err := tx.Where("id = ? AND khao_sat_id = ?", ans.CauHoiID, surveyID).
				First(&q).Error; err != nil {
				return fmt.Errorf("không tìm thấy câu hỏi %d: %w", ans.CauHoiID, err)
			}

			ct := models.CauTraLoi{
				PhanHoiID: submission.ID,
				CauHoiID:  ans.CauHoiID,
			}

			switch strings.ToUpper(q.LoaiCauHoi) {
			case "MULTIPLE_CHOICE":
				ct.LuaChon = ans.LuaChon
			case "UPLOAD_FILE":
				fileKey := fmt.Sprintf("file_%d", ans.CauHoiID)
				fileHeader, err := c.FormFile(fileKey)
				if err != nil {
					return fmt.Errorf("thiếu file bắt buộc cho câu hỏi %d: %w", ans.CauHoiID, err)
				}

				// Validate file
				if err := validateFile(fileHeader); err != nil {
					return fmt.Errorf("file không hợp lệ cho câu hỏi %d: %w", ans.CauHoiID, err)
				}

				fileID := fmt.Sprintf("%d_%d", submission.ID, ans.CauHoiID)
				folder := "answers"

				publicURL, upErr := utils.UploadToSupabase(
					fileHeader,
					fileHeader.Filename,
					fileID,
					folder,
					"",
				)
				if upErr != nil {
					return fmt.Errorf("upload thất bại cho câu hỏi %d: %w", ans.CauHoiID, upErr)
				}

				ct.NoiDung = publicURL

			default:
				ct.NoiDung = ans.NoiDung
			}

			if err := tx.Create(&ct).Error; err != nil {
				return fmt.Errorf("không lưu câu trả lời %d: %w", ans.CauHoiID, err)
			}
		}

		// 11. Cập nhật số phản hồi
		return tx.Model(&models.KhaoSat{}).
			Where("id = ?", surveyID).
			UpdateColumn("so_phan_hoi", gorm.Expr("so_phan_hoi + 1")).Error
	})

	if err != nil {
		log.Printf("Lỗi khi lưu phản hồi: %v", err)
		c.JSON(http.StatusInternalServerError,
			gin.H{"error": "Không thể lưu phản hồi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gửi khảo sát thành công"})
}

// Helper functions
func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func validateFile(fileHeader *multipart.FileHeader) error {
	// Giới hạn kích thước file (10MB)
	if fileHeader.Size > 10<<20 {
		return fmt.Errorf("file vượt quá kích thước cho phép")
	}

	// Kiểm tra loại file (chỉ cho phép một số loại nhất định)
	allowedTypes := map[string]bool{
		"image/jpeg":         true,
		"image/png":          true,
		"image/gif":          true,
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}

	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	// Chỉ kiểm tra 512 byte đầu để xác định loại file
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}

	contentType := http.DetectContentType(buffer)
	if !allowedTypes[contentType] {
		return fmt.Errorf("loại file không được hỗ trợ")
	}

	return nil
}

// GET /api/forms/:id/submissions?page=1&limit=10&start_date=2025-09-01&end_date=2025-09-21
func GetSubmissions(c *gin.Context) {
	// Parse survey ID
	surveyIDStr := c.Param("id")
	surveyID, err := strconv.Atoi(surveyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID khảo sát không hợp lệ"})
		return
	}

	// Kiểm tra khảo sát tồn tại
	var ks models.KhaoSat
	if err := config.DB.First(&ks, surveyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Khảo sát không tồn tại"})
		return
	}

	// Query params: pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Query params: filter date
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	query := config.DB.Model(&models.PhanHoi{}).
		Where("khao_sat_id = ?", surveyID)

	// Nếu có start_date
	if startDateStr != "" {
		if startDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
			query = query.Where("ngay_gui >= ?", startDate)
		}
	}

	// Nếu có end_date
	if endDateStr != "" {
		if endDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// endDate + 1 day để inclusive
			query = query.Where("ngay_gui < ?", endDate.Add(24*time.Hour))
		}
	}

	// Đếm tổng submissions
	var total int64
	query.Count(&total)

	// Lấy dữ liệu kèm preload
	var submissions []models.PhanHoi
	if err := query.
		Preload("NguoiDung").
		Preload("CauTraLois").
		Order("ngay_gui DESC").
		Limit(limit).Offset(offset).
		Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách phản hồi"})
		return
	}

	// Format response
	resp := []gin.H{}
	for _, s := range submissions {
		answers := []gin.H{}
		for _, a := range s.CauTraLois {
			answers = append(answers, gin.H{
				"cau_hoi_id": a.CauHoiID,
				"noi_dung":   a.NoiDung,
				"lua_chon":   a.LuaChon,
			})
		}

		resp = append(resp, gin.H{
			"id":       s.ID,
			"email":    s.Email,
			"user_id":  s.NguoiDungID,
			"user":     s.NguoiDung,
			"ngay_gui": s.NgayGui,
			"lan_gui":  s.LanGui,
			"answers":  answers,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"form_id":     surveyID,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"submissions": resp,
	})
}

// GET /api/forms/:id/submissions/:sub_id
func GetSubmissionDetail(c *gin.Context) {
	// Parse form ID
	formIDStr := c.Param("id")
	formID, err := strconv.Atoi(formIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID khảo sát không hợp lệ"})
		return
	}

	// Parse submission ID
	subIDStr := c.Param("sub_id")
	subID, err := strconv.Atoi(subIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phản hồi không hợp lệ"})
		return
	}

	// Kiểm tra khảo sát tồn tại
	var ks models.KhaoSat
	if err := config.DB.First(&ks, formID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Khảo sát không tồn tại"})
		return
	}

	// Lấy chi tiết submission
	var submission models.PhanHoi
	if err := config.DB.
		Preload("NguoiDung").
		Preload("CauTraLois").
		Where("id = ? AND khao_sat_id = ?", subID, formID).
		First(&submission).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Phản hồi không tồn tại"})
		return
	}

	// Chuẩn hoá response
	answers := []gin.H{}
	for _, a := range submission.CauTraLois {
		answers = append(answers, gin.H{
			"cau_hoi_id": a.CauHoiID,
			"noi_dung":   a.NoiDung,
			"lua_chon":   a.LuaChon,
		})
	}

	resp := gin.H{
		"id":       submission.ID,
		"form_id":  submission.KhaoSatID,
		"email":    submission.Email,
		"user_id":  submission.NguoiDungID,
		"user":     submission.NguoiDung,
		"ngay_gui": submission.NgayGui,
		"lan_gui":  submission.LanGui,
		"answers":  answers,
	}

	c.JSON(http.StatusOK, resp)
}

// BE-26-1: Dashboard thống kê phản hồi
func GetFormDashboard(c *gin.Context) {
	formID := c.Param("id")
	db := config.DB

	var questions []models.CauHoi
	if err := db.Where("khao_sat_id = ?", formID).Find(&questions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy câu hỏi"})
		return
	}

	results := []gin.H{}

	for _, q := range questions {
		stat := gin.H{
			"question_id": q.ID,
			"type":        strings.ToUpper(q.LoaiCauHoi),
			"content":     q.NoiDung,
			"stats":       nil,
		}

		switch strings.ToUpper(q.LoaiCauHoi) {
		// -----------------------------
		// Các loại câu hỏi chọn đáp án
		case "SINGLE_CHOICE", "MULTIPLE_CHOICE", "TRUE_FALSE":
			var rows []struct {
				Option string
				Count  int
			}
			db.Raw(`
		SELECT lua_chon AS option, COUNT(*) AS count
		FROM cau_tra_loi
		WHERE cau_hoi_id = ?
		GROUP BY lua_chon
	`, q.ID).Scan(&rows)

			// tính tổng để lấy %
			var total int
			for _, r := range rows {
				total += r.Count
			}

			stats := []gin.H{}
			for _, r := range rows {
				var parsed []string
				value := r.Option

				// Thử parse JSON array (["Cam"])
				if err := json.Unmarshal([]byte(r.Option), &parsed); err == nil {
					if len(parsed) == 1 {
						value = parsed[0] // lấy "Cam"
					} else {
						// nếu nhiều giá trị, trả luôn cả mảng
						b, _ := json.Marshal(parsed)
						value = string(b)
					}
				}

				stats = append(stats, gin.H{
					"option":  value,
					"count":   r.Count,
					"percent": float64(r.Count) * 100 / float64(total),
				})
			}
			stat["stats"] = stats

		// -----------------------------
		// Rating
		case "RATING":
			var rows []struct {
				Rating int
				Count  int
			}
			db.Raw(`
				SELECT noi_dung::INTEGER AS rating, COUNT(*) AS count
				FROM cau_tra_loi
				WHERE cau_hoi_id = $1
				AND noi_dung IS NOT NULL
				AND noi_dung <> ''
				GROUP BY noi_dung::INTEGER
				ORDER BY rating;
			`, q.ID).Scan(&rows)

			var sum, total int
			min, max := math.MaxInt, 0
			stats := []gin.H{}
			for _, r := range rows {
				stats = append(stats, gin.H{"rating": r.Rating, "count": r.Count})
				sum += r.Rating * r.Count
				total += r.Count
				if r.Rating < min {
					min = r.Rating
				}
				if r.Rating > max {
					max = r.Rating
				}
			}

			if total == 0 {
				stat["stats"] = gin.H{"avg": 0, "min": 0, "max": 0, "histogram": []gin.H{}}
			} else {
				stat["stats"] = gin.H{
					"avg":       float64(sum) / float64(total),
					"min":       min,
					"max":       max,
					"histogram": stats,
				}
			}

		// -----------------------------
		// Điền văn bản
		case "FILL_BLANK":
			var rows []struct {
				Answer string
				Count  int
			}
			db.Raw(`
				SELECT noi_dung AS answer, COUNT(*) AS count
				FROM cau_tra_loi
				WHERE cau_hoi_id = ?
				GROUP BY noi_dung
			`, q.ID).Scan(&rows)

			stats := []gin.H{}
			for _, r := range rows {
				stats = append(stats, gin.H{
					"answer": r.Answer,
					"count":  r.Count,
				})
			}
			stat["stats"] = stats

		// -----------------------------
		// Upload file
		case "UPLOAD_FILE":
			var rows []struct {
				UserID sql.NullInt64
				File   sql.NullString
			}
			db.Raw(`
				SELECT ph.nguoi_dung_id AS user_id, ctl.noi_dung AS file
				FROM cau_tra_loi ctl
				JOIN phan_hoi ph ON ctl.phan_hoi_id = ph.id
				WHERE ctl.cau_hoi_id = $1
				AND ctl.noi_dung IS NOT NULL
				AND ctl.noi_dung <> '';
			`, q.ID).Scan(&rows)

			files := []gin.H{}
			for _, r := range rows {
				files = append(files, gin.H{
					"user_id": r.UserID.Int64,
					"file":    r.File.String,
				})
			}
			stat["stats"] = files
		}

		results = append(results, stat)
	}

	c.JSON(http.StatusOK, gin.H{
		"form_id": formID,
		"results": results,
	})
}
