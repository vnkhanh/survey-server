package models

import (
	"time"
)

type Answer struct {
	ID        uint    `gorm:"primaryKey"`
	KhaoSatID uint    `json:"khao_sat_id"`          // Liên kết với khảo sát
	KhaoSat   KhaoSat `gorm:"foreignKey:KhaoSatID"` // Quan hệ với KhaoSat
	CauTraLoi string  `json:"cau_tra_loi"`
	CreatedAt time.Time
}
