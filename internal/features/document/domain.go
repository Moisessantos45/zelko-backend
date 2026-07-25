package document

import (
	"context"
	"fmt"
	"strings"
	"zelko/internal/models"
)

type Paginated struct {
	Data     []models.Document `json:"data"`
	Paginate models.Pagination `json:"paginate"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	List(ctx context.Context, userID uint64, offset, limit int) ([]models.Document, int64, error)
	GetBySlug(ctx context.Context, slug string) (*models.Document, error)
	IsPublicBySlug(ctx context.Context, slug string) (bool, error)
	ListVisibility(ctx context.Context) ([]models.Visibility, error)
	IsOwner(ctx context.Context, userID uint64, id uint64) (bool, error)
	IsOwnerBySlug(ctx context.Context, userID uint64, slug string) (bool, error)
	Create(ctx context.Context, data *models.Document) error
	Update(ctx context.Context, slug string, data map[string]any) error
}

type Service interface {
	List(ctx context.Context, authID uint64, page int, pageSize int) (*Paginated, error)
	GetBySlug(ctx context.Context, authID uint64, slug string) (*models.Document, error)
	IsPublicBySlug(ctx context.Context, slug string) (bool, error)
	ListVisibility(ctx context.Context) ([]models.Visibility, error)
	Create(ctx context.Context, authID uint64, title string, content string) (*models.Document, error)
	Update(ctx context.Context, authID uint64, slug string, title *string, content *string) error
	UpdateVisibility(ctx context.Context, authID uint64, slug string, visibilityID uint64) error
}

func New(userID uint64, title string, content string) (*models.Document, error) {
	if userID == 0 {
		return nil, fmt.Errorf("invalid userID: %d", userID)
	}

	if strings.TrimSpace(title) == "" {
		return nil, fmt.Errorf("invalid title: %s", title)
	}

	return &models.Document{
		UserID:  userID,
		Title:   title,
		Content: content,
	}, nil
}

func BuildUpdateData(title *string, content *string) map[string]any {
	updateData := make(map[string]any)

	if title != nil && strings.TrimSpace(*title) != "" {
		updateData["title"] = *title
	}

	if content != nil && strings.TrimSpace(*content) != "" {
		updateData["content"] = *content
	}

	return updateData
}

func generateURL(host string, slug string, visibilityID uint64) string {
	if visibilityID == 2 {
		return fmt.Sprintf("%s/api/v1/documents/%s/preview", host, slug)
	}
	return fmt.Sprintf("%s/api/v1/documents/public/%s/preview", host, slug)
}
