package models

type CauTraLoi struct {
	ID        uint     `gorm:"primaryKey;autoIncrement" json:"id"`
	PhanHoiID uint     `json:"phan_hoi_id"`
	PhanHoi   PhanHoi  `gorm:"foreignKey:PhanHoiID;constraint:OnDelete:CASCADE" json:"phan_hoi"`
	CauHoiID  uint     `json:"cau_hoi_id"`
	CauHoi    CauHoi   `gorm:"foreignKey:CauHoiID;constraint:OnDelete:CASCADE" json:"cau_hoi"`
	NoiDung   string   `gorm:"type:text" json:"noi_dung"`
	LuaChonID *uint    `json:"lua_chon_id"`
	LuaChon   *LuaChon `gorm:"foreignKey:LuaChonID" json:"lua_chon"`
}
