package document

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"zelko/internal/shared/errorsx"
	"zelko/internal/shared/helper"
	"zelko/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type CreateRequest struct {
	Title        string `json:"title" binding:"required"`
	Content      string `json:"content" binding:"required"`
	TypeDocument string `json:"type_document" binding:"required"`
}

type UpdateRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

type Handler struct {
	uc Service
}

func NewHandler(uc Service) *Handler {
	return &Handler{
		uc: uc,
	}
}
func jwtError(c *gin.Context, err error) {
	log.Printf("Error extrayendo datos del JWT: %v", err)
	helper.RespondError(c, errorsx.ErrUnauthorized)
}

func (h *Handler) ListVisibility(c *gin.Context) {
	visibilities, err := h.uc.ListVisibility(c.Request.Context())
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"data": visibilities, "message": "Visibilities listed successfully"})
}

func (h *Handler) List(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	queryParams, err := utils.ParsePagination(c)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	results, err := h.uc.List(c.Request.Context(), authID, queryParams.Page, queryParams.PageSize)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"data": results, "message": "Documents listed successfully"})
}

func (h *Handler) GetBySlug(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		helper.RespondError(c, errorsx.ErrNotFound)
		return
	}

	document, err := h.uc.GetBySlug(c.Request.Context(), authID, slug)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"data": document, "message": "Document retrieved successfully"})
}

func (h *Handler) Create(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	var req CreateRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	document, err := h.uc.Create(c.Request.Context(), authID, req.Title, req.Content, req.TypeDocument)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(201, gin.H{"data": document, "message": "Document created successfully"})
}

func (h *Handler) Update(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		helper.RespondError(c, errorsx.ErrNotFound)
		return
	}

	var req UpdateRequest
	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.uc.Update(c.Request.Context(), authID, slug, req.Title, req.Content)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Document updated successfully"})
}

func (h *Handler) UpdateVisibility(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		helper.RespondError(c, errorsx.ErrNotFound)
		return
	}

	var req struct {
		VisibilityID uint64 `json:"visibility_id" binding:"required"`
	}

	if !helper.BindJSON(c, &req) {
		return
	}

	err = h.uc.UpdateVisibility(c.Request.Context(), authID, slug, req.VisibilityID)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	c.JSON(200, gin.H{"message": "Document visibility updated successfully"})
}

func (h *Handler) Compile(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		helper.RespondError(c, errorsx.ErrNotFound)
		return
	}

	document, err := h.uc.GetBySlug(c.Request.Context(), authID, slug)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	if document.Content == "" {
		helper.RespondError(c, errorsx.ErrInvalidData)
		return
	}

	baseDir, err := utils.AppDir()
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	engine := strings.ToLower(strings.TrimSpace(document.TypeDocument))
	if engine == "" {
		engine = "latex" // documentos antiguos creados antes de existir type_document
	}

	if err := ValidateSource(engine, document.Content); err != nil {
		helper.RespondError(c, err)
		return
	}

	srcName := "main.tex"
	if engine == "typst" {
		srcName = "main.typ"
	}

	jobDir := filepath.Join(baseDir, "tmp", slug)

	_ = os.RemoveAll(jobDir)

	if err := os.MkdirAll(jobDir, 0750); err != nil {
		helper.RespondError(c, err)
		return
	}

	if err := os.WriteFile(filepath.Join(jobDir, srcName), []byte(document.Content), 0640); err != nil {
		helper.RespondError(c, err)
		return
	}

	logOut, err := RunCompiler(engine, jobDir, srcName)
	if err != nil {
		helper.RespondError(c, err)
		utils.CleanupDirLater(jobDir, 5*time.Minute)
		return
	}

	pdfPath := filepath.Join(jobDir, "main.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		helper.RespondError(c, err)
		utils.CleanupDirLater(jobDir, 5*time.Minute)
		return
	}

	utils.CleanupDirLater(jobDir, 10*time.Minute)

	c.JSON(http.StatusOK, gin.H{
		"ok":     true,
		"slug":   slug,
		"pdfUrl": fmt.Sprintf("/api/v1/documents/%s/preview", slug),
		"log":    logOut,
	})
}

func (h *Handler) Preview(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		jwtError(c, err)
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		helper.RespondError(c, errorsx.ErrNotFound)
		return
	}

	if _, err := h.uc.GetBySlug(c.Request.Context(), authID, slug); err != nil {
		helper.RespondError(c, err)
		return
	}

	baseDir, err := utils.AppDir()
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	pdfPath := filepath.Join(baseDir, "tmp", slug, "main.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		helper.RespondError(c, err)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `inline; filename="main.pdf"`)
	c.Header("Cache-Control", "no-store")
	c.File(pdfPath)
}

func (h *Handler) PreviewPublic(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		helper.RespondError(c, errorsx.ErrNotFound)
		return
	}

	_, err := h.uc.IsPublicBySlug(c.Request.Context(), slug)
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	baseDir, err := utils.AppDir()
	if err != nil {
		helper.RespondError(c, err)
		return
	}

	pdfPath := filepath.Join(baseDir, "tmp", slug, "main.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		helper.RespondError(c, err)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `inline; filename="main.pdf"`)
	c.Header("Cache-Control", "no-store")
	c.File(pdfPath)
}
