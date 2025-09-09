package models

import "time"

type NguoiDung struct {
	ID      uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Ten     string    `gorm:"column:ten;size:100;not null" json:"ten"`
	Email   string    `gorm:"column:email;size:100;unique;not null" json:"email"`
	MatKhau string    `gorm:"column:mat_khau;size:255;not null" json:"mat_khau"`
	NgayTao time.Time `gorm:"column:ngay_tao;autoCreateTime" json:"ngay_tao"`
	VaiTro  bool      `gorm:"column:vai_tro;not null;default:false" json:"vai_tro"`

	// Quan hệ
	KhaoSats     []KhaoSat          `gorm:"foreignKey:NguoiTaoID" json:"-"`
	PhanHois     []PhanHoi          `gorm:"foreignKey:NguoiDungID" json:"-"`
	Rooms        []Room             `gorm:"foreignKey:NguoiTaoID" json:"-"`
	RoomThamGias []RoomNguoiThamGia `gorm:"foreignKey:NguoiDungID" json:"-"`
}

func (NguoiDung) TableName() string {
	return "nguoi_dung"
}
