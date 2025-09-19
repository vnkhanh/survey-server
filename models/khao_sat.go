package models

import "time"

type KhaoSat struct {
	ID            uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TieuDe        string     `gorm:"column:tieu_de;size:255;not null" json:"tieu_de"`
	MoTa          string     `gorm:"column:mo_ta;type:text" json:"mo_ta"`
	TrangThai     string     `gorm:"column:trang_thai;size:20;default:'draft'" json:"trang_thai"`
	NguoiTaoID    *uint      `gorm:"column:nguoi_tao_id" json:"nguoi_tao_id"`
	NgayTao       time.Time  `gorm:"column:ngay_tao;autoCreateTime" json:"ngay_tao"`
	NgayKetThuc   *time.Time `gorm:"column:ngay_ket_thuc" json:"ngay_ket_thuc"`
	TemplateID    *uint      `gorm:"column:template_id" json:"template_id"`
	SoPhanHoi     int        `gorm:"column:so_phan_hoi" json:"so_phan_hoi"`
	SettingsJSON  string     `gorm:"column:settings_json;type:text" json:"-"`
	ThemeJSON     string     `gorm:"column:theme_json;type:text" json:"-"`
	EditTokenHash string     `gorm:"column:edit_token_hash;type:text" json:"-"`
	// Thêm 2 trường mới để lưu public link và embed code
	PublicLink *string `gorm:"column:public_link;size:255" json:"public_link"`
	EmbedCode  *string `gorm:"column:embed_code;type:text" json:"embed_code"`

	NguoiTao *NguoiDung `gorm:"foreignKey:NguoiTaoID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`

	// Quan hệ
	CauHois  []CauHoi  `gorm:"foreignKey:KhaoSatID" json:"-"`
	PhanHois []PhanHoi `gorm:"foreignKey:KhaoSatID" json:"-"`
	Rooms    []Room    `gorm:"foreignKey:KhaoSatID" json:"-"`
}

func (KhaoSat) TableName() string {
	return "khao_sat"
}
