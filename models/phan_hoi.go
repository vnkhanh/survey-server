package models

import "time"

type PhanHoi struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	KhaoSatID   uint      `gorm:"column:khao_sat_id;not null;index" json:"khao_sat_id"`
	NguoiDungID *uint     `gorm:"column:nguoi_dung_id;index" json:"nguoi_dung_id"`
	NgayGui     time.Time `gorm:"column:ngay_gui;autoCreateTime" json:"ngay_gui"`
	LanGui      int       `gorm:"column:lan_gui;default:1" json:"lan_gui"`
	Email       *string   `gorm:"column:email;size:100" json:"email"`

	// Quan hệ
	KhaoSat    *KhaoSat    `gorm:"foreignKey:KhaoSatID" json:"-"`
	NguoiDung  *NguoiDung  `gorm:"foreignKey:NguoiDungID" json:"-"`
	CauTraLois []CauTraLoi `gorm:"foreignKey:PhanHoiID" json:"-"`
}

func (PhanHoi) TableName() string {
	return "phan_hoi"
}
