package models

type CauHoi struct {
	ID         uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	KhaoSatID  uint   `gorm:"column:khao_sat_id;not null" json:"khao_sat_id"`
	NoiDung    string `gorm:"column:noi_dung;type:text;not null" json:"noi_dung"`
	LoaiCauHoi string `gorm:"column:loai_cau_hoi;size:50;not null" json:"loai_cau_hoi"`
	ThuTu      int    `gorm:"column:thu_tu;default:0" json:"thu_tu"`

	// Quan hệ
	LuaChons   []LuaChon   `gorm:"foreignKey:CauHoiID" json:"-"`
	CauTraLois []CauTraLoi `gorm:"foreignKey:CauHoiID" json:"-"`
}

func (CauHoi) TableName() string {
	return "cau_hoi"
}
