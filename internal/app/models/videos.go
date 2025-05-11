package models

import "time"

type Video struct {
	ID        uint64    `gorm:"primaryKey"`
	UserID    uint64    `gorm:"column:user_id;not null"`
	URL       string    `gorm:"column:url;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
