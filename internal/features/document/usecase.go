package document

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"zelko/internal/features/user"
	"zelko/internal/models"
	"zelko/internal/shared/utils"
)

type UseCase struct {
	repo   Repository
	authUc user.Service
}

func NewUseCase(repo Repository, authUc user.Service) Service {
	return &UseCase{repo: repo, authUc: authUc}
}

func (uc *UseCase) List(ctx context.Context, authID uint64, page int, pageSize int) (*Paginated, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	userID, err := uc.authUc.GetIDByAuthID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID by authID: %w", err)
	}

	if page <= 0 {
		page = 1
	}

	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	results, total, err := uc.repo.List(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if total > 0 && page > totalPages {
		return nil, errors.New("page out of range")
	}

	return &Paginated{
		Data: results,
		Paginate: models.Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil

}

func (uc *UseCase) GetBySlug(ctx context.Context, authID uint64, slug string) (*models.Document, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("invalid slug: %s", slug)
	}

	userID, err := uc.authUc.GetIDByAuthID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID by authID: %w", err)
	}

	isOwner, err := uc.repo.IsOwnerBySlug(ctx, userID, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check document ownership: %w", err)
	}

	if !isOwner {
		return nil, fmt.Errorf("user with authID %d is not the owner of document with slug %s", authID, slug)
	}

	document, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get document by slug: %w", err)
	}

	return document, nil
}

func (uc *UseCase) ListVisibility(ctx context.Context) ([]models.Visibility, error) {
	visibilities, err := uc.repo.ListVisibility(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list visibilities: %w", err)
	}
	return visibilities, nil
}

func (uc *UseCase) IsPublicBySlug(ctx context.Context, slug string) (bool, error) {
	if strings.TrimSpace(slug) == "" {
		return false, fmt.Errorf("invalid slug: %s", slug)
	}

	isPublic, err := uc.repo.IsPublicBySlug(ctx, slug)
	if err != nil {
		return false, fmt.Errorf("failed to check if document is public: %w", err)
	}

	if !isPublic {
		return false, fmt.Errorf("document with slug %s is not public", slug)
	}

	return isPublic, nil
}

func (uc *UseCase) Create(ctx context.Context, authID uint64, title string, content string, typeDocument string) (*models.Document, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	userID, err := uc.authUc.GetIDByAuthID(ctx, authID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID by authID: %w", err)
	}

	newDocument, err := New(userID, title, content, typeDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to create new document: %w", err)
	}

	newDocument.Slug = utils.GenerateObjectIDLike()
	newDocument.VisibilityID = 2

	host := os.Getenv("HOST_API_PROD")
	if host == "" {
		host = "http://localhost:4106"
	}
	
	newDocument.URL = generateURL(host, newDocument.Slug, newDocument.VisibilityID)

	err = uc.repo.Create(ctx, newDocument)
	if err != nil {
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	return newDocument, nil
}

func (uc *UseCase) Update(ctx context.Context, authID uint64, slug string, title *string, content *string) error {
	if authID == 0 {
		return fmt.Errorf("invalid authID: %d", authID)
	}

	userID, err := uc.authUc.GetIDByAuthID(ctx, authID)
	if err != nil {
		return fmt.Errorf("failed to get user ID by authID: %w", err)
	}

	isOwner, err := uc.repo.IsOwnerBySlug(ctx, userID, slug)
	if err != nil {
		return fmt.Errorf("failed to check document ownership: %w", err)
	}

	if !isOwner {
		return fmt.Errorf("user with authID %d is not the owner of document with ID %s", authID, slug)
	}

	updateData := BuildUpdateData(title, content)
	if len(updateData) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	err = uc.repo.Update(ctx, slug, updateData)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return nil
}

func (uc *UseCase) UpdateVisibility(ctx context.Context, authID uint64, slug string, visibilityID uint64) error {
	if authID == 0 {
		return fmt.Errorf("invalid authID: %d", authID)
	}

	if visibilityID < 1 || visibilityID > 3 {
		return fmt.Errorf("invalid visibility_id: %d, must be 1 (public), 2 (private), or 3 (shared)", visibilityID)
	}

	userID, err := uc.authUc.GetIDByAuthID(ctx, authID)
	if err != nil {
		return fmt.Errorf("failed to get user ID by authID: %w", err)
	}

	isOwner, err := uc.repo.IsOwnerBySlug(ctx, userID, slug)
	if err != nil {
		return fmt.Errorf("failed to check document ownership: %w", err)
	}

	if !isOwner {
		return fmt.Errorf("user with authID %d is not the owner of document with slug %s", authID, slug)
	}

	host := os.Getenv("HOST_API_PROD")
	if host == "" {
		host = "http://localhost:4106"
	}

	url := generateURL(host, slug, visibilityID)

	updateData := map[string]any{
		"visibility_id": visibilityID,
		"url":           url,
	}

	err = uc.repo.Update(ctx, slug, updateData)
	if err != nil {
		return fmt.Errorf("failed to update document visibility: %w", err)
	}

	return nil
}
