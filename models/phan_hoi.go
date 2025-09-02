package models

import "time"

type PhanHoi struct {
	ID          uint `gorm:"primaryKey;autoIncrement"`
	KhaoSatID   uint
	KhaoSat     KhaoSat `gorm:"foreignKey:KhaoSatID;constraint:OnDelete:CASCADE"`
	NguoiDungID *uint
	NguoiDung   *NguoiDung  `gorm:"foreignKey:NguoiDungID"`
	NgayGui     time.Time   `gorm:"autoCreateTime"`
	CauTraLois  []CauTraLoi `gorm:"foreignKey:PhanHoiID"`
}
