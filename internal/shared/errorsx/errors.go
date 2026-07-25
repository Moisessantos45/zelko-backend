package errorsx

import "errors"

var (
	ErrCodeAccountInactive  = errors.New("ACCOUNT_INACTIVE")
	ErrCodeEmailNotVerified = errors.New("EMAIL_NOT_VERIFIED")
	ErrEmailNotVerified     = errors.New("email no verificado")
	ErrPending2FA           = errors.New("PENDING_2FA")
	ErrNotFound             = errors.New("resource not found")
	ErrAlreadyExists        = errors.New("resource already exists")
	ErrInvalidReference     = errors.New("invalid reference")
	ErrInvalidData          = errors.New("invalid data")
	ErrTimeout              = errors.New("operation timed out")
	ErrCanceled             = errors.New("operation canceled")
)
