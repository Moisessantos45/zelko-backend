package routes

import (
	"zelko/internal/features/user"

	"github.com/gin-gonic/gin"
)

func UserRoutes(rg *gin.RouterGroup) {
	h := user.NewUserHandler(userUc)

	protected := rg.Group("/user")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.GET("/me", h.GetByAuthID)
		protected.POST("", h.Create)
		protected.PUT("/me", h.Update)
		protected.POST("/me/upload-avatar", h.UploadAvatar)
	}
}
