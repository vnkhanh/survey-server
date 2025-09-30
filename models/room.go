package models

import "time"

type Room struct {
	ID          uint               `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	KhaoSatID   uint               `gorm:"column:khao_sat_id;not null" json:"khao_sat_id"`
	TenRoom     string             `gorm:"column:ten_room;size:100;not null" json:"ten_room"`
	MoTa        *string            `gorm:"column:mo_ta;type:text" json:"mo_ta"`
	MatKhau     *string            `gorm:"column:mat_khau;size:255" json:"-"`
	NguoiTaoID  *uint              `gorm:"column:nguoi_tao_id" json:"nguoi_tao_id"`
	TrangThai   string             `gorm:"column:trang_thai;size:20;default:'active'" json:"trang_thai"`
	IsPublic    *bool              `gorm:"column:is_public;default:true" json:"is_public"`
	Khoa        bool               `gorm:"column:khoa;default:false" json:"khoa"`
	NgayTao     time.Time          `gorm:"column:ngay_tao;autoCreateTime" json:"ngay_tao"`
	NgayCapNhat time.Time          `gorm:"column:ngay_cap_nhat;autoUpdateTime" json:"ngay_cap_nhat"`
	ShareURL    string             `gorm:"column:share_url;size:255" json:"share_url"`
	Members     []RoomNguoiThamGia `json:"members" gorm:"foreignKey:RoomID"`
	ThamGias    []RoomNguoiThamGia `gorm:"foreignKey:RoomID" json:"-"`
	IsLocked    bool               `gorm:"column:is_locked;default:false" json:"is_locked"`
	KhaoSat     KhaoSat            `gorm:"foreignKey:KhaoSatID;references:ID" json:"khao_sat"`
}

func (Room) TableName() string {
	return "room"
}
