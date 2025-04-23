package repository

import (
	"errors"
	"gorm.io/gorm"
	"log/slog"
	"shakalizator/internal/app/models"
	"shakalizator/pkg/logger"
)

type VideosRepository struct {
	log *logger.Logger
	db  *gorm.DB
}

func NewVideos(log *logger.Logger, db *gorm.DB) *VideosRepository {
	return &VideosRepository{
		log: log,
		db:  db,
	}
}

func (r *VideosRepository) Create(video *models.Video) error {
	if err := r.db.Create(video).Error; err != nil {
		r.log.Error("failed to create video", err)
		return err
	}
	return nil
}

func (r *VideosRepository) Get(userId uint64, url string) (*models.Video, error) {
	var video models.Video

	err := r.db.Where("user_id = ? AND url = ?", userId, url).First(&video).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Error("failed to get video", err, slog.Uint64("userID", userId), slog.String("url", url))
		}
		return nil, err
	}

	return &video, nil
}

func (r *VideosRepository) Update(video *models.Video) error {
	result := r.db.Save(video)
	if result.Error != nil {
		r.log.Error("failed to update video", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *VideosRepository) Delete(url string) error {
	result := r.db.Where("url = ?", url).Delete(&models.Video{})
	if result.Error != nil {
		r.log.Error("failed to delete video", result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *VideosRepository) GetByUserID(userID string) ([]models.Video, error) {
	var videos []models.Video
	if err := r.db.Where("user_id = ?", userID).Find(&videos).Error; err != nil {
		r.log.Error("failed to get videos by user ID", err)
		return nil, err
	}
	return videos, nil
}
