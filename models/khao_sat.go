package models

import "time"

type KhaoSat struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	TieuDe      string `gorm:"size:255;not null"`
	MoTa        string `gorm:"type:text"`
	TrangThai   string `gorm:"size:20;default:draft"`
	NguoiTaoID  uint
	NguoiTao    NguoiDung `gorm:"foreignKey:NguoiTaoID;constraint:OnDelete:CASCADE"`
	NgayTao     time.Time `gorm:"autoCreateTime"`
	NgayKetThuc *time.Time
	CauHois     []CauHoi  `gorm:"foreignKey:KhaoSatID"`
	PhanHois    []PhanHoi `gorm:"foreignKey:KhaoSatID"`
}
