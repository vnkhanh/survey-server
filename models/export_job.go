package models

import "time"

type ExportJob struct {
    JobID              string     `gorm:"column:job_id;primaryKey;size:36" json:"job_id"`
    KhaoSatID          uint       `gorm:"column:khao_sat_id;index" json:"khao_sat_id"`
    Format             string     `gorm:"column:format;size:10" json:"format"` // csv, xlsx
    RangeFrom          *time.Time `gorm:"column:range_from" json:"range_from,omitempty"`
    RangeTo            *time.Time `gorm:"column:range_to" json:"range_to,omitempty"`
    IncludeAttachments bool       `gorm:"column:include_attachments" json:"include_attachments"`
    Status             string     `gorm:"column:status;size:20;default:'queued'" json:"status"`
    FilePath           *string    `gorm:"column:file_path;type:text" json:"file_path,omitempty"`
    ErrorMsg           *string    `gorm:"column:error_msg;type:text" json:"error_msg,omitempty"`
    CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ExportJob) TableName() string {
    return "export_jobs"
}
