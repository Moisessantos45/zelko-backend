package user

import (
	"context"
	"zelko/internal/models"
	"zelko/internal/shared/errorsx"
	"zelko/internal/shared/helper"

	"errors"

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

func (r *PostgresRepository) GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	var userID uint64
	err := r.db.WithContext(ctx).Model(&models.User{}).Select("id").Where("auth_id = ?", authID).Scan(&userID).Error
	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return 0, nil
		}
		return 0, helper.TranslateDBError(err)
	}

	return userID, nil
}

func (r *PostgresRepository) GetByAuthID(ctx context.Context, authID uint64) (*Detail, error) {
	var user Detail

	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select(`
			users.id,
			users.first_name,
			users.last_name,
			auths.email
		`).
		Joins("JOIN auths ON auths.id = users.auth_id").
		Where("users.auth_id = ?", authID).
		First(&user).Error

	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return nil, nil
		}
		return nil, helper.TranslateDBError(err)
	}

	return &user, nil
}

func (r *PostgresRepository) GetEmailByID(ctx context.Context, id uint64) (string, error) {
	var email string
	err := r.db.WithContext(ctx).
		Model(&models.User{}).
		Select("auths.email").
		Joins("JOIN auths ON auths.id = users.auth_id").
		Where("users.id = ?", id).
		Scan(&email).Error
	if err != nil {
		if errors.Is(helper.TranslateDBError(err), errorsx.ErrNotFound) {
			return "", nil
		}
		return "", helper.TranslateDBError(err)
	}

	return email, nil
}

func (r *PostgresRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *PostgresRepository) Update(ctx context.Context, id uint64, data map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Updates(data)

	if result.Error != nil {
		return helper.TranslateDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return errorsx.ErrNotFound
	}

	return nil
}

func (r *PostgresRepository) UpdateAvatarURL(ctx context.Context, userID uint64, url string) error {
	return helper.TranslateDBError(r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("avatar_url", url).Error)
}
