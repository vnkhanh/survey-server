package models

type CauHoi struct {
	ID         uint `gorm:"primaryKey;autoIncrement"`
	KhaoSatID  uint
	KhaoSat    KhaoSat     `gorm:"foreignKey:KhaoSatID;constraint:OnDelete:CASCADE"`
	NoiDung    string      `gorm:"type:text;not null"`
	LoaiCauHoi string      `gorm:"size:50;not null"`
	ThuTu      int         `gorm:"default:0"`
	LuaChons   []LuaChon   `gorm:"foreignKey:CauHoiID"`
	CauTraLois []CauTraLoi `gorm:"foreignKey:CauHoiID"`
}
