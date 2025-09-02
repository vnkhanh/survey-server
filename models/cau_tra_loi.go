package models

type CauTraLoi struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	PhanHoiID uint
	PhanHoi   PhanHoi `gorm:"foreignKey:PhanHoiID;constraint:OnDelete:CASCADE"`
	CauHoiID  uint
	CauHoi    CauHoi `gorm:"foreignKey:CauHoiID;constraint:OnDelete:CASCADE"`
	NoiDung   string `gorm:"type:text"`
	LuaChonID *uint
	LuaChon   *LuaChon `gorm:"foreignKey:LuaChonID"`
}
