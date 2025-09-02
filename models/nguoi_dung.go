package models

import "time"

type NguoiDung struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	Ten      string    `gorm:"size:100;not null"`
	Email    string    `gorm:"size:100;unique;not null"`
	MatKhau  string    `gorm:"size:255;not null"`
	NgayTao  time.Time `gorm:"autoCreateTime"`
	VaiTro   bool      `gorm:"not null;default:false"`
	KhaoSats []KhaoSat `gorm:"foreignKey:NguoiTaoID"`
	PhanHois []PhanHoi `gorm:"foreignKey:NguoiDungID"`
}
