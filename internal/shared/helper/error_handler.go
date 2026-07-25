package helper

import (
	"context"
	"zelko/internal/shared/errorsx"

	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

func TranslateDBError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return errorsx.ErrNotFound
	case errors.Is(err, context.Canceled):
		return errorsx.ErrCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return errorsx.ErrTimeout
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23505":
			return errorsx.ErrAlreadyExists
		case "23503":
			return errorsx.ErrInvalidReference
		case "23502":
			return errorsx.ErrInvalidData
		}
	}

	return fmt.Errorf("repository operation failed: %w", err)
}
