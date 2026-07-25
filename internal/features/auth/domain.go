package auth

import (
	"context"
	"zelko/internal/models"
	"zelko/internal/shared/utils"

	"fmt"

	"strings"
)

type UserBasic struct {
	ID    string `gorm:"column:id"`
	Email string `gorm:"column:email"`
}

type Repository interface {
	WithTransaction(fn func(repo *PostgresRepository) error) error
	Register(user *models.Auth) error
	GetByEmail(email string) (*models.Auth, error)
	GetByEmailAny(email string) (*models.Auth, error)
	ExistsByEmail(email string) (bool, error)
	CountUsers() (int64, error)
	GetActiveUserByEmail(email string) (*models.Auth, error)
	GetByID(id uint64) (*models.Auth, error)
	GetBasicByID(id uint64) (*UserBasic, error)
	GetBasicByEmail(email string) (*UserBasic, error)

	UpdateEmailConfirm(id uint64) error
	UpdateEmail(id uint64, email string) error
	UpdatePassword(id uint64, newPassword string) error
	SetTwoFactor(id uint64, enabled bool, secret string) error
	ChangeCompleteProfile(id uint64, complete bool) error
	ChangeStatus(id uint64, active bool) error
	DeleteByID(id uint64) error
}

type Service interface {
	Register(ctx context.Context, email string, password string) (*models.Auth, error)
	Login(ctx context.Context, email string, password string) (*models.Auth, string, error)
	GetSession(userId uint64) (*models.Auth, error)
	GetBasicByID(id uint64) (*UserBasic, error)
	RefreshToken(ctx context.Context, token string) (string, error)
	VerifyEmail(email string) error
	ExistsByEmail(email string) (bool, error)
	ForwardEmailVerification(ctx context.Context, email string) error
	ConfirmAccount(ctx context.Context, userId uint64, token string) error
	SendPasswordReset(ctx context.Context, email string) error
	ResetPassword(userId uint64, newPassword string) error
	UpdatePassword(userId uint64, currentPassword string, newPassword string) error
	UpdateEmail(ctx context.Context, authID uint64, newEmail string) error
	UpdateCompleteProfile(userId uint64) error
	ChangeCompleteProfile(userId uint64, complete bool) error
	Deactivate(userId uint64) error
	EnableTWOFA(userId uint64) (string, string, error)
	VerifyTOTP(userId uint64, code string) (bool, error)
	DisableTWOFA(userId uint64) error
	Verify2FALogin(ctx context.Context, sessionID string, code string) (*models.Auth, string, error)
	Logout(ctx context.Context, token string) error
}

func NewUser(email string, password string) (*models.Auth, error) {
	if email == "" || len(strings.TrimSpace(email)) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 || len(password) > 24 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 6 caracteres y no más de 24 caracteres")
	}

	return &models.Auth{
		Email:        email,
		PasswordHash: password,
		FullProfile:  false,
	}, nil
}

func NewUserAdmin(email string, password string) (*models.Auth, error) {
	if email == "" || len(strings.TrimSpace(email)) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 || len(password) > 24 {
		return nil, fmt.Errorf("la contraseña debe tener al menos 6 caracteres y no más de 24 caracteres")
	}

	return &models.Auth{
		Email:          email,
		PasswordHash:   password,
		FullProfile:    false,
		EmailConfirmed: false,
		FirstSession:   false,
	}, nil
}
