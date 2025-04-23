package repository

import (
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"shakalizator/internal/app/models"
	"shakalizator/pkg/logger"
)

type UsersRepository struct {
	log *logger.Logger
	db  *gorm.DB
}

func NewUsers(log *logger.Logger, db *gorm.DB) *UsersRepository {
	return &UsersRepository{
		log: log,
		db:  db,
	}
}

func (r *UsersRepository) GetViolations(userID uint64) (int, error) {
	var user models.User
	err := r.db.Select("violations").First(&user, userID).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("failed to get user violations", err, slog.Uint64("userID", userID))
		}
		return 0, err
	}
	return user.Violations, nil
}

func (r *UsersRepository) IncrementViolations(userID uint64) error {
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("violations", gorm.Expr("violations + 1"))

	if result.Error != nil {
		r.log.Error("failed to increment violations", result.Error, slog.Uint64("userID", userID))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UsersRepository) ResetViolations(userID uint64) error {
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("violations", 0)

	if result.Error != nil {
		r.log.Error("failed to reset violations", result.Error, slog.Uint64("userID", userID))
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UsersRepository) Create(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		r.log.Error("failed to create user", err)
		return err
	}
	return nil
}
