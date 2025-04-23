package models

type User struct {
	ID         uint64 `gorm:"primaryKey"`
	Violations int    `gorm:"not null"`
}
