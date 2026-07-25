package routes

import (
	"os"
	"zelko/config"
	"zelko/config/db"
	"zelko/internal/features/auth"
	"zelko/internal/features/document"
	"zelko/internal/features/user"
	"zelko/internal/shared/middleware"
	"zelko/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

var (
	maker  *utils.PasetoMaker
	authUc auth.Service
	userUc user.Service
	docUc  document.Service
)

func Init() {
	rd := config.Rdb
	maker = utils.NewPasetoMaker()

	frontendURL := os.Getenv("HOST_URL_PROD")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	aRp := auth.NewPostgresRepository(db.DB)
	authUc = auth.NewAuthUseCase(aRp, rd, maker, frontendURL)

	uRp := user.NewPostgresRepository(db.DB)
	userUc = user.NewUserUseCase(uRp, authUc)

	dRp := document.NewPostgresRepository(db.DB)
	docUc = document.NewUseCase(dRp, userUc)

}

func authMiddleware() gin.HandlerFunc {
	return middleware.AuthMiddleware(maker, config.Rdb)
}

func storageAuthMiddleware() gin.HandlerFunc {
	return middleware.StorageAuthMiddleware(maker, config.Rdb)
}

func preAuthMiddleware() gin.HandlerFunc {
	return middleware.PreAuthMiddleware(config.Rdb)
}

func idempotencyMiddleware() gin.HandlerFunc {
	return middleware.IdempotencyMiddleware(config.Rdb)
}
