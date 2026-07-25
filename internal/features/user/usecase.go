package user

import (
	"context"
	"fmt"
	"log"
	"zelko/internal/features/auth"
	"zelko/internal/models"
)

type UserUseCase struct {
	repo   Repository
	ucAuth auth.Service
}

func NewUserUseCase(repo Repository, ucAuth auth.Service) Service {
	return &UserUseCase{repo: repo, ucAuth: ucAuth}
}

func (uc *UserUseCase) GetIDByAuthID(ctx context.Context, authID uint64) (uint64, error) {
	if authID == 0 {
		return 0, nil
	}

	return uc.repo.GetIDByAuthID(ctx, authID)
}

func (uc *UserUseCase) GetByAuthID(ctx context.Context, authID uint64) (*Detail, error) {
	if authID == 0 {
		return nil, nil
	}

	user, err := uc.repo.GetByAuthID(ctx, authID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (uc *UserUseCase) GetEmailByID(ctx context.Context, id uint64) (string, error) {
	if id == 0 {
		return "", nil
	}

	return uc.repo.GetEmailByID(ctx, id)
}

func (uc *UserUseCase) Create(ctx context.Context, authID uint64, firstName, lastName string) (*models.User, error) {
	if authID == 0 {
		return nil, fmt.Errorf("invalid authID: %d", authID)
	}

	newUser, err := New(authID, firstName, lastName)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return nil, err
	}

	log.Printf("Creating user with AuthID: %d", newUser.AuthID)

	err = uc.repo.WithTransaction(func(repo *PostgresRepository) error {
		if err := uc.repo.Create(ctx, newUser); err != nil {
			return err
		}

		if err := uc.ucAuth.ChangeCompleteProfile(newUser.AuthID, true); err != nil {
			return fmt.Errorf("error al actualizar el perfil completo en auth: %v", err)
		}

		return nil
	})
	log.Printf("User created successfully with ID: %d", newUser.ID)

	return newUser, err
}

func (uc *UserUseCase) Update(ctx context.Context, authID uint64, firstName *string, lastName *string) error {
	if authID == 0 {
		return nil
	}

	user, err := uc.repo.GetByAuthID(ctx, authID)
	if err != nil {
		return err
	}

	if user == nil {
		return fmt.Errorf("user not found for authID: %d", authID)
	}

	updateData := BuildUpdateData(firstName, lastName)

	return uc.repo.Update(ctx, user.ID, updateData)
}

func (uc *UserUseCase) UploadAvatar(ctx context.Context, authID uint64, fileId, path, mimeType string, size int64) error {
	// user, err := uc.repo.GetByAuthID(ctx, authID)
	// if err != nil {
	// 	return err
	// }

	// if user == nil {
	// 	return nil
	// }

	// return uc.repo.WithTransaction(func(txRepo *PostgresRepository) error {
	// 	file := &models.File{
	// 		FileID:       fileId,
	// 		Path:         path,
	// 		MimeType:     mimeType,
	// 		Size:         size,
	// 		PermissionID: 1,
	// 		UserID:       user.ID,
	// 	}
	// 	if err := txRepo.CreateFile(ctx, file); err != nil {
	// 		return err
	// 	}

	// 	url := utils.GenerateURL("storage", fileId)
	// 	return txRepo.UpdateAvatarURL(ctx, user.ID, url)
	// })
	return nil
}
