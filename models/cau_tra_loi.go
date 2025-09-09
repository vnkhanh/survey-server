package models

type CauTraLoi struct {
	ID        uint    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	PhanHoiID uint    `gorm:"column:phan_hoi_id;not null" json:"phan_hoi_id"`
	CauHoiID  uint    `gorm:"column:cau_hoi_id;not null" json:"cau_hoi_id"`
	NoiDung   *string `gorm:"column:noi_dung;type:text" json:"noi_dung"`
	LuaChonID *uint   `gorm:"column:lua_chon_id" json:"lua_chon_id"`
}

func (CauTraLoi) TableName() string {
	return "cau_tra_loi"
}
