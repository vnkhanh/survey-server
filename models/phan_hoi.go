package models

import "time"

type PhanHoi struct {
	ID          uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	KhaoSatID   uint        `json:"khao_sat_id"`
	KhaoSat     KhaoSat     `gorm:"foreignKey:KhaoSatID;constraint:OnDelete:CASCADE" json:"khao_sat"`
	NguoiDungID *uint       `json:"nguoi_dung_id"`
	NguoiDung   *NguoiDung  `gorm:"foreignKey:NguoiDungID" json:"nguoi_dung"`
	NgayGui     time.Time   `gorm:"autoCreateTime" json:"ngay_gui"`
	CauTraLois  []CauTraLoi `gorm:"foreignKey:PhanHoiID" json:"cau_tra_loi"`
}
