package auth

import (
	"context"
	"zelko/internal/models"
	"zelko/internal/shared/errorsx"
	"zelko/internal/shared/templates"
	"zelko/internal/shared/utils"

	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type AuthUseCase struct {
	rd   *redis.Client
	repo Repository
	mk   *utils.PasetoMaker
	host string
}

func NewAuthUseCase(repo Repository, rd *redis.Client, mk *utils.PasetoMaker, host string) Service {
	return &AuthUseCase{
		repo: repo,
		rd:   rd,
		mk:   mk,
		host: host,
	}
}

func (a *AuthUseCase) createSession(ctx context.Context, user *models.Auth) (string, error) {
	sessionID, err := utils.GenerateSessionID()
	if err != nil {
		return "", err
	}

	sessionData := utils.NewSession(user)
	key := fmt.Sprintf("session:%s", sessionID)
	ttl := 8 * time.Hour

	err = a.rd.HSet(ctx, key, sessionData).Err()
	if err != nil {
		return "", err
	}

	err = a.rd.Expire(ctx, key, ttl).Err()
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (a *AuthUseCase) Register(ctx context.Context, email string, password string) (*models.Auth, error) {
	existingUser, err := a.repo.ExistsByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("no se pudo completar el registro, intente más tarde")
	}

	if existingUser {
		return nil, fmt.Errorf("el correo electrónico ya se encuentra registrado")
	}

	newUser, err := NewUser(email, password)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := utils.HashPassword(newUser.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	newUser.PasswordHash = hashedPassword

	err = a.repo.WithTransaction(func(txRepo *PostgresRepository) error {
		if err := txRepo.Register(newUser); err != nil {
			return fmt.Errorf("No se pudo registrar el usuario")
		}

		if newUser.ID == 0 {
			return fmt.Errorf("No se pudo registrar el usuario")
		}

		token, err := a.mk.NewToken(fmt.Sprintf("%d", newUser.ID), 15*time.Minute)
		if err != nil {
			return fmt.Errorf("Error generating token")
		}

		confirmKey := fmt.Sprintf("confirm:%d", newUser.ID)
		err = a.rd.Set(ctx, confirmKey, token, 15*time.Minute).Err()
		if err != nil {
			return fmt.Errorf("error caching token: %w", err)
		}

		renderer, err := templates.NewEmailRenderer()
		if err != nil {
			return err
		}

		data := templates.ConfirmAccountData{
			Name:        newUser.Email,
			ConfirmLink: fmt.Sprintf("%s/confirm/%s", a.host, token),
		}

		htmlContent, err := renderer.RenderConfirmAccount(data)
		if err != nil {
			return err
		}

		err = utils.EnqueueEmail([]string{newUser.Email}, "Confirmacion de cuenta", htmlContent)
		if err != nil {
			return fmt.Errorf("error sending email: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return newUser, nil
}

func (a *AuthUseCase) Login(ctx context.Context, email string, password string) (*models.Auth, string, error) {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return nil, "", fmt.Errorf("email inválido: %s", email)
	}

	if password == "" || len(password) < 6 {
		return nil, "", fmt.Errorf("la contraseña debe tener al menos 6 caracteres")
	}

	user, err := a.repo.GetActiveUserByEmail(email)
	if err != nil {
		if errors.Is(err, errorsx.ErrCodeEmailNotVerified) {
			return nil, "", errorsx.ErrCodeEmailNotVerified
		}

		return nil, "", fmt.Errorf("No se encontro email")
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, "", fmt.Errorf("contraseña incorrecta")
	}

	if user.TwoFactorEnabled {
		sessionID, err := utils.GenerateSessionID()
		if err != nil {
			return nil, "", fmt.Errorf("error generando sesión temporal")
		}

		key := fmt.Sprintf("preauth:%s", sessionID)
		preAuthData := map[string]any{
			"user_id":            fmt.Sprintf("%d", user.ID),
			"two_factor_enabled": true,
			"auth_stage":         "pending_2fa",
			"mfa_verified":       false,
		}

		if err := a.rd.HSet(ctx, key, preAuthData).Err(); err != nil {
			return nil, "", fmt.Errorf("error guardando sesión temporal: %w", err)
		}

		if err := a.rd.Expire(ctx, key, 5*time.Minute).Err(); err != nil {
			return nil, "", fmt.Errorf("error configurando TTL de sesión temporal: %w", err)
		}

		return nil, sessionID, errorsx.ErrPending2FA
	}

	accessTTL := 15 * time.Minute
	refreshTTL := 10 * time.Hour

	accessToken, err := a.mk.NewToken(fmt.Sprintf("%d", user.ID), accessTTL)
	if err != nil {
		return nil, "", err
	}

	refreshToken, err := a.mk.NewToken(fmt.Sprintf("%d", user.ID), refreshTTL)
	if err != nil {
		return nil, "", err
	}

	_, err = a.createSession(ctx, user)
	if err != nil {
		return nil, "", err
	}

	user.Token = accessToken
	user.PasswordHash = ""

	return user, refreshToken, nil
}

func (a *AuthUseCase) Verify2FALogin(ctx context.Context, sessionID string, code string) (*models.Auth, string, error) {
	if sessionID == "" {
		return nil, "", fmt.Errorf("sesión temporal inválida")
	}

	key := fmt.Sprintf("preauth:%s", sessionID)

	vals, err := a.rd.HGetAll(ctx, key).Result()
	if err != nil || len(vals) == 0 {
		return nil, "", fmt.Errorf("sesión temporal expirada o inválida")
	}

	authStage, ok := vals["auth_stage"]
	if !ok || authStage != "pending_2fa" {
		return nil, "", fmt.Errorf("estado de sesión inválido")
	}

	userIDStr, ok := vals["user_id"]
	if !ok {
		return nil, "", fmt.Errorf("sesión temporal corrupta")
	}

	var userID uint64
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		return nil, "", fmt.Errorf("ID de usuario inválido en sesión")
	}

	user, err := a.repo.GetByID(userID)
	if err != nil {
		return nil, "", err
	}

	if !utils.VerifyTOTP(user.TwoFactorSecret, code) {
		return nil, "", fmt.Errorf("código 2FA inválido")
	}

	a.rd.Del(ctx, key)

	accessToken, err := a.mk.NewToken(fmt.Sprintf("%d", user.ID), 15*time.Minute)
	if err != nil {
		return nil, "", fmt.Errorf("error generando access token")
	}

	refreshToken, err := a.mk.NewToken(fmt.Sprintf("%d", user.ID), 10*time.Hour)
	if err != nil {
		return nil, "", fmt.Errorf("error generando refresh token")
	}

	_, err = a.createSession(ctx, user)
	if err != nil {
		return nil, "", err
	}

	user.Token = accessToken
	user.PasswordHash = ""

	return user, refreshToken, nil
}

func (a *AuthUseCase) VerifyEmail(email string) error {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return fmt.Errorf("email inválido: %s", email)
	}

	exists, err := a.repo.ExistsByEmail(email)
	if err != nil {
		return fmt.Errorf("error verificando email: %w", err)
	}
	if !exists {
		return fmt.Errorf("No se encontro email")
	}

	return nil
}

func (a *AuthUseCase) ExistsByEmail(email string) (bool, error) {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return false, fmt.Errorf("email inválido: %s", email)
	}

	exists, err := a.repo.ExistsByEmail(email)
	if err != nil {
		return false, fmt.Errorf("error checking email existence: %w", err)
	}

	return exists, nil
}

func (a *AuthUseCase) GetSession(userId uint64) (*models.Auth, error) {
	if userId == 0 {
		return nil, fmt.Errorf("El ID de usuario no puede ser cero")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	user.PasswordHash = ""

	return user, nil
}

func (a *AuthUseCase) GetBasicByID(id uint64) (*UserBasic, error) {
	if id == 0 {
		return nil, fmt.Errorf("El ID de usuario no puede ser cero")
	}

	return a.repo.GetBasicByID(id)
}

func (a *AuthUseCase) RefreshToken(ctx context.Context, token string) (string, error) {
	payload, err := a.mk.VerifyToken(token)
	if err != nil {
		return "", fmt.Errorf("Invalid token: %w", err)
	}

	userId := payload.UserID

	var authID uint64
	fmt.Sscanf(userId, "%d", &authID)

	newToken, err := a.mk.NewToken(userId, 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("error generating new token: %w", err)
	}

	return newToken, nil
}

func (a *AuthUseCase) ForwardEmailVerification(ctx context.Context, email string) error {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return fmt.Errorf("email inválido: %s", email)
	}

	user, err := a.repo.GetByEmailAny(email)
	if err != nil {
		return fmt.Errorf("No se encontro email")
	}
	if user == nil {
		return fmt.Errorf("No se encontro email")
	}

	token, err := a.mk.NewToken(fmt.Sprintf("%d", user.ID), 15*time.Minute)
	if err != nil {
		return err
	}

	confirmKey := fmt.Sprintf("confirm:%d", user.ID)
	if err := a.rd.Set(ctx, confirmKey, token, 15*time.Minute).Err(); err != nil {
		return fmt.Errorf("error caching token: %w", err)
	}

	renderer, err := templates.NewEmailRenderer()
	if err != nil {
		return err
	}

	data := templates.ConfirmAccountData{
		Name:        user.Email,
		ConfirmLink: fmt.Sprintf("%s/confirm/%s", a.host, token),
	}

	htmlContent, err := renderer.RenderConfirmAccount(data)
	if err != nil {
		return err
	}

	err = utils.EnqueueEmail([]string{email}, "Confirmacion de cuenta", htmlContent)
	if err != nil {
		return fmt.Errorf("No se pudo enviar el correo electrónico de confirmación: %w", err)
	}

	return nil
}

func (a *AuthUseCase) ConfirmAccount(ctx context.Context, userId uint64, token string) error {
	if userId == 0 {
		return fmt.Errorf("El ID de usuario no puede ser cero")
	}

	if token == "" {
		return fmt.Errorf("token inválido")
	}

	confirmKey := fmt.Sprintf("confirm:%d", userId)

	storedToken, err := a.rd.Get(ctx, confirmKey).Result()
	if err != nil {
		return fmt.Errorf("token expirado o inválido")
	}

	if storedToken != token {
		return fmt.Errorf("token inválido")
	}

	if err := a.repo.UpdateEmailConfirm(userId); err != nil {
		return fmt.Errorf("No se pudo confirmar el correo electrónico")
	}

	a.rd.Del(ctx, confirmKey)

	return nil
}

func (a *AuthUseCase) SendPasswordReset(ctx context.Context, email string) error {
	if email == "" || len(email) <= 10 || !utils.EmailRegex.MatchString(email) {
		return fmt.Errorf("email inválido: %s", email)
	}

	user, err := a.repo.GetBasicByEmail(email)
	if err != nil {
		log.Printf("Password reset requested for non-existent email: %s", email)
		return nil
	}

	token, err := a.mk.NewToken(fmt.Sprintf("%s", user.ID), 15*time.Minute)
	if err != nil {
		return err
	}

	renderer, err := templates.NewEmailRenderer()
	if err != nil {
		return err
	}

	data := templates.PasswordResetData{
		Name:      user.Email,
		ResetLink: fmt.Sprintf("%s/forgot-password/%s", a.host, token),
	}

	htmlContent, err := renderer.RenderPasswordReset(data)
	if err != nil {
		return err
	}

	err = utils.EnqueueEmail([]string{email}, "Restablece tu contraseña", htmlContent)
	if err != nil {
		return fmt.Errorf("No se pudo enviar el correo electrónico de restablecimiento de contraseña: %w", err)
	}

	return nil
}

func (a *AuthUseCase) ResetPassword(userId uint64, newPassword string) error {
	if newPassword == "" || len(newPassword) < 6 {
		return fmt.Errorf("la contraseña debe tener al menos 6 caracteres")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("No se pudo restablecer la contraseña")
	}

	err = a.repo.UpdatePassword(userId, hashedPassword)
	if err != nil {
		return fmt.Errorf("No se pudo restablecer la contraseña")
	}

	return nil
}

func (a *AuthUseCase) UpdatePassword(userId uint64, currentPassword string, newPassword string) error {
	if userId == 0 {
		return fmt.Errorf("El ID de usuario no puede ser cero")
	}

	if newPassword == "" || len(newPassword) < 6 {
		return fmt.Errorf("la contraseña debe tener al menos 6 caracteres")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return fmt.Errorf("No se encontro el usuario")
	}

	if !utils.CheckPasswordHash(currentPassword, user.PasswordHash) {
		return fmt.Errorf("contraseña actual incorrecta")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("No se pudo actualizar la contraseña")
	}

	err = a.repo.UpdatePassword(userId, hashedPassword)
	if err != nil {
		return fmt.Errorf("No se pudo actualizar la contraseña")
	}

	return nil
}

func (a *AuthUseCase) ChangeCompleteProfile(userId uint64, complete bool) error {
	if userId == 0 {
		return fmt.Errorf("El ID de usuario no puede ser cero")
	}

	err := a.repo.ChangeCompleteProfile(userId, complete)
	if err != nil {
		return fmt.Errorf("No se pudo actualizar el perfil")
	}

	return nil
}

func (a *AuthUseCase) Deactivate(userId uint64) error {
	if userId == 0 {
		return fmt.Errorf("El ID de usuario no puede ser cero")
	}

	err := a.repo.ChangeStatus(userId, false)
	if err != nil {
		return fmt.Errorf("No se pudo desactivar el usuario")
	}

	return nil
}

func (a *AuthUseCase) EnableTWOFA(userId uint64) (string, string, error) {
	if userId == 0 {
		return "", "", fmt.Errorf("El ID de usuario no puede ser cero")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return "", "", fmt.Errorf("El usuario no existe")
	}

	uri, secret, err := utils.GenerateTOTP(user.Email, "SpinLuck")
	if err != nil {
		return "", "", fmt.Errorf("No se pudo generar el secreto de 2FA")
	}

	err = a.repo.SetTwoFactor(userId, true, secret)
	if err != nil {
		return "", "", fmt.Errorf("No se pudo habilitar la autenticación de dos factores")
	}

	return uri, secret, nil
}

func (a *AuthUseCase) VerifyTOTP(userId uint64, code string) (bool, error) {
	if userId == 0 {
		return false, fmt.Errorf("El ID de usuario no puede ser cero")
	}

	user, err := a.repo.GetByID(userId)
	if err != nil {
		return false, fmt.Errorf("El usuario no existe")
	}

	if !user.TwoFactorEnabled {
		return false, fmt.Errorf("La autenticación de dos factores no está habilitada para este usuario")
	}

	return utils.VerifyTOTP(user.TwoFactorSecret, code), nil
}

func (a *AuthUseCase) DisableTWOFA(userId uint64) error {
	if userId == 0 {
		return fmt.Errorf("El ID de usuario no puede ser cero")
	}

	err := a.repo.SetTwoFactor(userId, false, "")
	if err != nil {
		return fmt.Errorf("No se pudo deshabilitar la autenticación de dos factores")
	}

	return nil
}

func (a *AuthUseCase) UpdateCompleteProfile(userId uint64) error {
	if userId == 0 {
		return fmt.Errorf("El ID de usuario no puede ser cero")
	}

	err := a.repo.ChangeCompleteProfile(userId, true)
	if err != nil {
		return fmt.Errorf("No se pudo actualizar el perfil")
	}

	return nil
}

func (a *AuthUseCase) Logout(ctx context.Context, token string) error {
	if token == "" || len(token) < 7 {
		return fmt.Errorf("missing or invalid token")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return nil
}

func (a *AuthUseCase) UpdateEmail(ctx context.Context, authID uint64, newEmail string) error {
	if authID == 0 {
		return fmt.Errorf("El ID de usuario no puede ser cero")
	}

	if newEmail == "" || len(newEmail) <= 10 || !utils.EmailRegex.MatchString(newEmail) {
		return fmt.Errorf("email inválido: %s", newEmail)
	}

	exists, err := a.repo.ExistsByEmail(newEmail)
	if err != nil {
		return fmt.Errorf("error checking email: %w", err)
	}

	if exists {
		return fmt.Errorf("el correo electrónico ya se encuentra registrado")
	}

	if err := a.repo.UpdateEmail(authID, newEmail); err != nil {
		return fmt.Errorf("No se pudo actualizar el correo electrónico")
	}

	if err := a.ForwardEmailVerification(ctx, newEmail); err != nil {
		return fmt.Errorf("se actualizó el email pero no se pudo enviar el correo de verificación: %w", err)
	}

	return nil
}
