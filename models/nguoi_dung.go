package models

import "time"

type NguoiDung struct {
	ID       uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Ten      string    `gorm:"size:100;not null" json:"ten"`
	Email    string    `gorm:"size:100;unique;not null" json:"email"`
	MatKhau  string    `gorm:"size:255;not null" json:"-"` // ẩn khi trả JSON
	NgayTao  time.Time `gorm:"autoCreateTime" json:"ngay_tao"`
	VaiTro   bool      `gorm:"not null;default:false" json:"vai_tro"`
	KhaoSats []KhaoSat `gorm:"foreignKey:NguoiTaoID" json:"khao_sat"`
	PhanHois []PhanHoi `gorm:"foreignKey:NguoiDungID" json:"phan_hoi"`
}
