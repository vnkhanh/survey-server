package models

type LuaChon struct {
	ID       uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CauHoiID uint   `gorm:"column:cau_hoi_id;not null" json:"cau_hoi_id"`
	NoiDung  string `gorm:"column:noi_dung;type:text;not null" json:"noi_dung"`
	ThuTu    int    `gorm:"column:thu_tu;default:0" json:"thu_tu"`
}

func (LuaChon) TableName() string {
	return "lua_chon"
}
