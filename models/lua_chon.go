package models

type LuaChon struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	CauHoiID uint   `json:"cau_hoi_id"`
	CauHoi   CauHoi `gorm:"foreignKey:CauHoiID;constraint:OnDelete:CASCADE" json:"cau_hoi"`
	NoiDung  string `gorm:"type:text;not null" json:"noi_dung"`
	ThuTu    int    `gorm:"default:0" json:"thu_tu"`
}
