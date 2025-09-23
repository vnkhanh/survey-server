package models

import "time"

type RoomNguoiThamGia struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RoomID      uint      `gorm:"column:room_id;not null;index" json:"room_id"`
	NguoiDungID uint      `gorm:"column:nguoi_dung_id;not null" json:"nguoi_dung_id"`
	NgayVao     time.Time `gorm:"column:ngay_vao;autoCreateTime" json:"ngay_vao"`
	TrangThai   string    `gorm:"column:trang_thai;size:20;default:'active'" json:"trang_thai"`
	IP          string    `gorm:"column:ip;size:45" json:"ip"`
	// Liên kết với Room, khi Room bị xóa thì xóa luôn record này
	Room Room `gorm:"foreignKey:RoomID;constraint:OnDelete:CASCADE;" json:"-"`
}

func (RoomNguoiThamGia) TableName() string {
	return "room_nguoi_tham_gia"
}
