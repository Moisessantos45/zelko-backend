package document

import (
	"context"
	"zelko/internal/models"
	"zelko/internal/shared/helper"

	"gorm.io/gorm"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) Repository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) WithTransaction(fn func(repo *PostgresRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &PostgresRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *PostgresRepository) ListVisibility(ctx context.Context) ([]models.Visibility, error) {
	var visibilities []models.Visibility

	if err := r.db.WithContext(ctx).Find(&visibilities).Error; err != nil {
		return nil, helper.TranslateDBError(err)
	}

	return visibilities, nil
}

func (r *PostgresRepository) List(ctx context.Context, userID uint64, offset, limit int) ([]models.Document, int64, error) {
	var documents []models.Document
	var total int64

	base := r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("user_id = ?", userID)

	queryCount := base.Session(&gorm.Session{})
	queryData := base.Session(&gorm.Session{})

	if err := queryCount.Count(&total).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	if err := queryData.Offset(offset).Limit(limit).
		Find(&documents).Error; err != nil {
		return nil, 0, helper.TranslateDBError(err)
	}

	return documents, total, nil
}

func (r *PostgresRepository) GetBySlug(ctx context.Context, slug string) (*models.Document, error) {
	var document models.Document
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&document).Error

	if err != nil {
		return nil, helper.TranslateDBError(err)
	}

	return &document, nil
}

func (r *PostgresRepository) IsPublicBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Document{}).
		Where("slug = ? AND visibility_id IN (1, 3)", slug).
		Count(&count).Error

	if err != nil {
		return false, helper.TranslateDBError(err)
	}

	return count > 0, nil
}

func (r *PostgresRepository) IsOwner(ctx context.Context, userID uint64, id uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Document{}).
		Where("id = ? AND user_id = ?", id, userID).
		Count(&count).Error

	if err != nil {
		return false, helper.TranslateDBError(err)
	}

	return count > 0, nil
}

func (r *PostgresRepository) IsOwnerBySlug(ctx context.Context, userID uint64, slug string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Document{}).
		Where("slug = ? AND user_id = ?", slug, userID).
		Count(&count).Error

	if err != nil {
		return false, helper.TranslateDBError(err)
	}

	return count > 0, nil
}

func (r *PostgresRepository) Create(ctx context.Context, data *models.Document) error {
	if err := r.db.WithContext(ctx).Create(data).Error; err != nil {
		return helper.TranslateDBError(err)
	}

	return nil
}

func (r *PostgresRepository) Update(ctx context.Context, slug string, data map[string]any) error {
	if err := r.db.WithContext(ctx).Model(&models.Document{}).
		Where("slug = ?", slug).Updates(data).Error; err != nil {
		return helper.TranslateDBError(err)
	}

	return nil
}
