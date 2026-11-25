package routes

import (
	"time"
	"zelko/config"
	"zelko/internal/features/auth"
	"zelko/internal/shared/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func AuthRoutes(rg *gin.RouterGroup) {
	s := authUc
	h := auth.NewAuthHandler(s)
	strict := middleware.RateLimiterRedis(config.Rdb, "auth", rate.Every(time.Minute/10), 5)

	public := rg.Group("")
	public.Use(idempotencyMiddleware())
	// public.Use(middleware.RateLimiterMiddleware(rate.Every(time.Minute/20), 20))
	{
		public.POST("/login", strict, h.Login)
		public.POST("/forward-email-verification", strict, h.ForwardEmailVerification)
		public.POST("/forgot-password", strict, h.SendPasswordReset)
		public.POST("/register", strict, h.Register)
		public.POST("/logout", h.Logout)
		public.POST("/refresh-token", h.RefreshToken)
	}

	protected := rg.Group("/")
	protected.Use(authMiddleware())
	protected.Use(idempotencyMiddleware())
	// protected.Use(middleware.RateLimiterMiddleware(rate.Every(time.Minute/300), 300))
	{
		protected.GET("/confirm-account", h.ConfirmAccount)
		protected.GET("/session", h.GetSession)
		protected.POST("/verify-email", h.VerifyEmail)
		protected.PATCH("/reset-password", h.ResetPassword)
		protected.PATCH("/change-password", h.UpdatePassword)

		protected.GET("/generate-two-factor", h.TestingCrateTWOFA)
	}
}
