package models

import "time"

type CauTraLoi struct {
	ID        uint `gorm:"primaryKey;autoIncrement" json:"id"`
	PhanHoiID uint `gorm:"column:phan_hoi_id;not null" json:"phan_hoi_id"`
	CauHoiID  uint `gorm:"column:cau_hoi_id;not null"  json:"cau_hoi_id"`

	// Dữ liệu trả lời:
	//  - fill_blank, rating, true_false, upload_file: lưu vào NoiDung
	//  - multiple_choice: lưu JSON mảng các lựa chọn
	NoiDung string `gorm:"column:noi_dung;type:text" json:"noi_dung"` // text, rating, bool, link file
	LuaChon string `gorm:"column:lua_chon;type:text" json:"lua_chon"` // JSON array string cho multiple_choice

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	PhanHoi *PhanHoi `gorm:"foreignKey:PhanHoiID" json:"-"`
	CauHoi  *CauHoi  `gorm:"foreignKey:CauHoiID"  json:"-"`
}

func (CauTraLoi) TableName() string {
	return "cau_tra_loi"
}
