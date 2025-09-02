package models

type LuaChon struct {
	ID       uint `gorm:"primaryKey;autoIncrement"`
	CauHoiID uint
	CauHoi   CauHoi `gorm:"foreignKey:CauHoiID;constraint:OnDelete:CASCADE"`
	NoiDung  string `gorm:"type:text;not null"`
	ThuTu    int    `gorm:"default:0"`
}
