package models

type Video struct {
	ID     uint64 `gorm:"primaryKey"`
	UserID uint64 `gorm:"column:user_id;not null"`
	URL    string `gorm:"column:url;not null"`
}
