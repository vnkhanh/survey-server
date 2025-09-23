package models

import "time"

type RoomInvite struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RoomID    uint      `gorm:"not null" json:"room_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	Email     string    `gorm:"size:255;not null" json:"email"`
	Status    string    `gorm:"size:20;default:'pending'" json:"status"` // pending | accepted | rejected
	CreatedAt time.Time `json:"created_at"`
}

func (RoomInvite) TableName() string {
	return "room_invites"
}
