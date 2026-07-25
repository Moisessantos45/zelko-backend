package routes

import (
	"zelko/internal/features/document"

	"github.com/gin-gonic/gin"
)

func DocumentRoutes(rg *gin.RouterGroup) {
	h := document.NewHandler(docUc)

	public := rg.Group("/documents")
	{
		public.GET("/public/:slug/preview", h.PreviewPublic)
	}

	protected := rg.Group("/documents")
	protected.Use(storageAuthMiddleware())
	protected.Use(idempotencyMiddleware())
	{
		protected.GET("", h.List)
		protected.GET("/visibilities", h.ListVisibility)
		protected.GET("/:slug", h.GetBySlug)
		protected.POST("", h.Create)
		protected.PUT("/:slug", h.Update)
		protected.PATCH("/:slug/visibility", h.UpdateVisibility)
		protected.POST("/:slug/compile", h.Compile)
		protected.GET("/:slug/preview", h.Preview)
	}
}
