package models

import "time"

type KhaoSat struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	TieuDe      string     `gorm:"size:255;not null" json:"tieu_de"`
	MoTa        string     `gorm:"type:text" json:"mo_ta"`
	TrangThai   string     `gorm:"size:20;default:draft" json:"trang_thai"`
	NguoiTaoID  uint       `json:"nguoi_tao_id"`
	NguoiTao    NguoiDung  `gorm:"foreignKey:NguoiTaoID;constraint:OnDelete:CASCADE" json:"nguoi_tao"`
	NgayTao     time.Time  `gorm:"autoCreateTime" json:"ngay_tao"`
	NgayKetThuc *time.Time `json:"ngay_ket_thuc"`
	CauHois     []CauHoi   `gorm:"foreignKey:KhaoSatID" json:"cau_hoi"`
	PhanHois    []PhanHoi  `gorm:"foreignKey:KhaoSatID" json:"phan_hoi"`
}
