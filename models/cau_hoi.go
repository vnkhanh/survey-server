package models

type CauHoi struct {
	ID         uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	KhaoSatID  uint        `json:"khao_sat_id"`
	KhaoSat    KhaoSat     `gorm:"foreignKey:KhaoSatID;constraint:OnDelete:CASCADE" json:"khao_sat"`
	NoiDung    string      `gorm:"type:text;not null" json:"noi_dung"`
	LoaiCauHoi string      `gorm:"size:50;not null" json:"loai_cau_hoi"`
	ThuTu      int         `gorm:"default:0" json:"thu_tu"`
	BatBuoc    bool        `gorm:"default:false" json:"bat_buoc"`
	LuaChons   []LuaChon   `gorm:"foreignKey:CauHoiID" json:"lua_chon"`
	CauTraLois []CauTraLoi `gorm:"foreignKey:CauHoiID" json:"cau_tra_loi"`
}
