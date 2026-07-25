package user

import (
	"context"
	"fmt"
	"strings"
	"zelko/internal/models"
)

type Detail struct {
	ID        uint64 `json:"id" gorm:"column:id"`
	FirstName string `json:"first_name" gorm:"column:first_name"`
	LastName  string `json:"last_name" gorm:"column:last_name"`
	Email     string `json:"email" gorm:"column:email"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetByAuthID(ctx context.Context, authID uint64) (*Detail, error)
	GetEmailByID(ctx context.Context, id uint64) (string, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, id uint64, data map[string]any) error
	UpdateAvatarURL(ctx context.Context, userID uint64, url string) error
}

type Service interface {
	GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error)
	GetByAuthID(ctx context.Context, authID uint64) (*Detail, error)
	GetEmailByID(ctx context.Context, id uint64) (string, error)
	Create(ctx context.Context, authID uint64, firstName, lastName string) (*models.User, error)
	Update(ctx context.Context, authID uint64, firstName *string, lastName *string) error
	UploadAvatar(ctx context.Context, authID uint64, fileId, path, mimeType string, size int64) error
}

func New(authID uint64, firstName string, lastName string) (*models.User, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	if strings.TrimSpace(firstName) == "" {
		return nil, fmt.Errorf("invalid firstName: %s", firstName)
	}

	if strings.TrimSpace(lastName) == "" {
		return nil, fmt.Errorf("invalid lastName: %s", lastName)
	}

	return &models.User{
		AuthID:    authID,
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

func BuildUpdateData(firstName *string, lastName *string) map[string]any {
	updateData := make(map[string]any)

	if firstName != nil && strings.TrimSpace(*firstName) != "" {
		updateData["first_name"] = *firstName
	}

	if lastName != nil && strings.TrimSpace(*lastName) != "" {
		updateData["last_name"] = *lastName
	}

	return updateData
}
