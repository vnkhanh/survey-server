package controllers

import (
    "encoding/csv"
    "fmt"
    "net/http"
    "os"
    "path"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
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

    filename := fmt.Sprintf("export_%s.csv", job.JobID)
    outPath := path.Join(outDir, filename)

    f, err := os.Create(outPath)
    if err != nil {
        em := err.Error()
        config.DB.Model(&job).Updates(map[string]interface{}{"status": "failed", "error_msg": em})
        return
    }
    defer f.Close()

    w := csv.NewWriter(f)
    defer w.Flush()

    header := []string{"response_id", "email", "ngay_gui", "lan_gui", "answers"}
    w.Write(header)

    var responses []models.PhanHoi
    q := config.DB.Preload("CauTraLois").Where("khao_sat_id = ?", job.KhaoSatID)
    if job.RangeFrom != nil {
        q = q.Where("ngay_gui >= ?", job.RangeFrom)
    }
    if job.RangeTo != nil {
        q = q.Where("ngay_gui <= ?", job.RangeTo)
    }
    if err := q.Find(&responses).Error; err != nil {
        em := err.Error()
        config.DB.Model(&job).Updates(map[string]interface{}{"status": "failed", "error_msg": em})
        return
    }

    for _, r := range responses {
        email := ""
        if r.Email != nil {
            email = *r.Email
        }
        answers := ""
        for _, a := range r.CauTraLois {
            answers += fmt.Sprintf("[%d:%s] ", a.CauHoiID, a.NoiDung)
        }
        row := []string{
            fmt.Sprintf("%d", r.ID),
            email,
            r.NgayGui.Format(time.RFC3339),
            fmt.Sprintf("%d", r.LanGui),
            answers,
        }
        w.Write(row)
    }

    fp := outPath
    config.DB.Model(&job).Updates(map[string]interface{}{"status": "done", "file_path": fp})
}
