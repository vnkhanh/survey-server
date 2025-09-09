package models

import "time"

type PhanHoi struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	KhaoSatID   uint      `gorm:"column:khao_sat_id;not null" json:"khao_sat_id"`
	NguoiDungID *uint     `gorm:"column:nguoi_dung_id" json:"nguoi_dung_id"`
	NgayGui     time.Time `gorm:"column:ngay_gui;autoCreateTime" json:"ngay_gui"`

	// Quan hệ
	CauTraLois []CauTraLoi `gorm:"foreignKey:PhanHoiID" json:"-"`
}

func (PhanHoi) TableName() string {
	return "phan_hoi"
}
