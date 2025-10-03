package controllers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/vnkhanh/survey-server/config"
	"github.com/vnkhanh/survey-server/models"
)

type ExportRequest struct {
	Format             string  `json:"format"`
	RangeFrom          *string `json:"range_from,omitempty"`
	RangeTo            *string `json:"range_to,omitempty"`
	IncludeAttachments bool    `json:"include_attachments"`
}

// POST /api/forms/:id/export
func CreateExport(c *gin.Context) {
	id := c.Param("id")

	var form models.KhaoSat
	if err := config.DB.First(&form, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "Form không tồn tại"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Lỗi DB"})
		return
	}

	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Payload không hợp lệ"})
		return
	}
	if req.Format == "" {
		req.Format = "csv"
	}

	var fromPtr, toPtr *time.Time
	if req.RangeFrom != nil {
		if t, err := time.Parse(time.RFC3339, *req.RangeFrom); err == nil {
			fromPtr = &t
		}
	}
	if req.RangeTo != nil {
		if t, err := time.Parse(time.RFC3339, *req.RangeTo); err == nil {
			toPtr = &t
		}
	}

	jobID := uuid.New().String()
	job := models.ExportJob{
		JobID:              jobID,
		KhaoSatID:          form.ID,
		Format:             req.Format,
		RangeFrom:          fromPtr,
		RangeTo:            toPtr,
		IncludeAttachments: req.IncludeAttachments,
		Status:             "queued",
	}
	config.DB.Create(&job)

	go processExportJob(jobID)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id": jobID,
		"status": "queued",
	})
}

// GET /api/exports/:job_id
func GetExport(c *gin.Context) {
	jobID := c.Param("job_id")
	var job models.ExportJob
	if err := config.DB.First(&job, "job_id = ?", jobID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "Job không tìm thấy"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Lỗi DB"})
		return
	}

	if job.Status == "done" && job.FilePath != nil {
		c.FileAttachment(*job.FilePath, path.Base(*job.FilePath))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id": job.JobID,
		"status": job.Status,
		"error":  job.ErrorMsg,
	})
}

// xử lý job xuất dữ liệu
func processExportJob(jobID string) {
	var job models.ExportJob
	if err := config.DB.First(&job, "job_id = ?", jobID).Error; err != nil {
		return
	}
	config.DB.Model(&job).Update("status", "processing")

	outDir := "./exports"
	os.MkdirAll(outDir, 0755)

	// Đặt tên file
	ext := strings.ToLower(job.Format)
	if ext != "xlsx" {
		ext = "csv"
	}
	filename := fmt.Sprintf("export_%s.%s", job.JobID, ext)
	outPath := path.Join(outDir, filename)

	// helper update fail
	failJob := func(em string) {
		config.DB.Model(&job).Updates(map[string]interface{}{
			"status":    "failed",
			"error_msg": em,
		})
	}

	// 1. Lấy danh sách câu hỏi
	var questions []models.CauHoi
	if err := config.DB.Where("khao_sat_id = ?", job.KhaoSatID).
		Order("id asc").Find(&questions).Error; err != nil {
		failJob(err.Error())
		return
	}

	// 2. Lấy danh sách phản hồi
	var responses []models.PhanHoi
	q := config.DB.Preload("CauTraLois").Where("khao_sat_id = ?", job.KhaoSatID)
	if job.RangeFrom != nil {
		q = q.Where("ngay_gui >= ?", job.RangeFrom)
	}
	if job.RangeTo != nil {
		q = q.Where("ngay_gui <= ?", job.RangeTo)
	}
	if err := q.Find(&responses).Error; err != nil {
		failJob(err.Error())
		return
	}

	// 3. Chuẩn bị header
	header := []string{"Dấu thời gian"}
	for _, q := range questions {
		if q.NoiDung != "" {
			header = append(header, q.NoiDung)
		} else {
			header = append(header, "Câu hỏi không có tiêu đề")
		}
	}

	// --- Nếu CSV ---
	if ext == "csv" {
		f, err := os.Create(outPath)
		if err != nil {
			failJob(err.Error())
			return
		}
		defer f.Close()

		// BOM UTF-8
		if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
			failJob(err.Error())
			return
		}

		w := csv.NewWriter(f)
		defer w.Flush()

		// Ghi header
		if err := w.Write(header); err != nil {
			failJob(err.Error())
			return
		}

		// Ghi dữ liệu
		for _, r := range responses {
			row := []string{r.NgayGui.Format("02/01/2006 15:04:05")}

			answerMap := make(map[uint]models.CauTraLoi)
			for _, a := range r.CauTraLois {
				answerMap[a.CauHoiID] = a
			}

			for _, q := range questions {
				val := ""
				if ans, ok := answerMap[q.ID]; ok {
					switch strings.ToUpper(q.LoaiCauHoi) {
					case "FILL_BLANK", "RATING":
						val = ans.NoiDung

					case "UPLOAD_FILE", "FILE_UPLOAD":
						if job.IncludeAttachments {
							val = ans.NoiDung
						} else if ans.NoiDung != "" {
							val = "[đã đính kèm]"
						}

					case "MULTIPLE_CHOICE", "TRUE_FALSE":
						if ans.LuaChon != "" {
							var opts []string
							if err := json.Unmarshal([]byte(ans.LuaChon), &opts); err == nil {
								val = strings.Join(opts, ", ")
							} else {
								val = ans.LuaChon
							}
						}
					}
				}
				row = append(row, val)
			}

			if err := w.Write(row); err != nil {
				failJob(err.Error())
				return
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			failJob(err.Error())
			return
		}

		// done
		config.DB.Model(&job).Updates(map[string]interface{}{
			"status":    "done",
			"file_path": outPath,
		})

	} else { // --- Nếu XLSX ---
		f := excelize.NewFile()
		sheet := f.GetSheetName(f.GetActiveSheetIndex())

		// header
		for i, h := range header {
			col, _ := excelize.ColumnNumberToName(i + 1)
			cell := fmt.Sprintf("%s1", col)
			f.SetCellValue(sheet, cell, h)
		}

		// data
		for ri, r := range responses {
			rowIdx := ri + 2
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx),
				r.NgayGui.Format("02/01/2006 15:04:05"))

			answerMap := make(map[uint]models.CauTraLoi)
			for _, a := range r.CauTraLois {
				answerMap[a.CauHoiID] = a
			}

			for qi, q := range questions {
				col, _ := excelize.ColumnNumberToName(qi + 2)
				cell := fmt.Sprintf("%s%d", col, rowIdx)

				val := ""
				if ans, ok := answerMap[q.ID]; ok {
					switch strings.ToUpper(q.LoaiCauHoi) {
					case "FILL_BLANK", "RATING":
						val = ans.NoiDung

					case "UPLOAD_FILE", "FILE_UPLOAD":
						if job.IncludeAttachments {
							val = ans.NoiDung
						} else if ans.NoiDung != "" {
							val = "[đã đính kèm]"
						}

					case "MULTIPLE_CHOICE", "TRUE_FALSE":
						if ans.LuaChon != "" {
							var opts []string
							if err := json.Unmarshal([]byte(ans.LuaChon), &opts); err == nil {
								val = strings.Join(opts, ", ")
							} else {
								val = ans.LuaChon
							}
						}
					}
					f.SetCellValue(sheet, cell, val)
				}
			}
		}

		if err := f.SaveAs(outPath); err != nil {
			failJob(err.Error())
			return
		}

		config.DB.Model(&job).Updates(map[string]interface{}{
			"status":    "done",
			"file_path": outPath,
		})
	}
}
