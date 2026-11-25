package document

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"zelko/internal/shared/helper"
	"zelko/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type CreateRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
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

func (h *Handler) ListVisibility(c *gin.Context) {
	visibilities, err := h.uc.ListVisibility(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to list visibilities: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": visibilities, "message": "Visibilities listed successfully"})
}

func (h *Handler) List(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
		return
	}

	queryParams, err := utils.ParsePagination(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid query parameters: " + err.Error()})
		return
	}

	results, err := h.uc.List(c.Request.Context(), authID, queryParams.Page, queryParams.PageSize)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to list documents: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": results, "message": "Documents listed successfully"})
}

func (h *Handler) GetBySlug(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	document, err := h.uc.GetBySlug(c.Request.Context(), authID, slug)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to get document: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": document, "message": "Document retrieved successfully"})
}

func (h *Handler) Create(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
		return
	}

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "Datos inválidos",
			"errors":  helper.HumanValidationErrors(err),
		})
		return
	}

	document, err := h.uc.Create(c.Request.Context(), authID, req.Title, req.Content)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to create document: " + err.Error()})
		return
	}

	c.JSON(201, gin.H{"data": document, "message": "Document created successfully"})
}

func (h *Handler) Update(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "Datos inválidos",
			"errors":  helper.HumanValidationErrors(err),
		})
		return
	}

	err = h.uc.Update(c.Request.Context(), authID, slug, req.Title, req.Content)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to update document: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Document updated successfully"})
}

func (h *Handler) UpdateVisibility(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	var req struct {
		VisibilityID uint64 `json:"visibility_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "Datos inválidos",
			"errors":  helper.HumanValidationErrors(err),
		})
		return
	}

	err = h.uc.UpdateVisibility(c.Request.Context(), authID, slug, req.VisibilityID)
	if err != nil {
		c.JSON(500, gin.H{"message": "Failed to update document visibility: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Document visibility updated successfully"})
}

func (h *Handler) Compile(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	document, err := h.uc.GetBySlug(c.Request.Context(), authID, slug)
	if err != nil {
		c.JSON(404, gin.H{"message": "Document not found: " + err.Error()})
		return
	}

	if document.Content == "" {
		c.JSON(400, gin.H{"ok": false, "error": "document has no content to compile"})
		return
	}

	baseDir, err := utils.AppDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	engine := strings.ToLower(c.DefaultQuery("engine", "latex"))

	if err := ValidateSource(engine, document.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "contenido rechazado: " + err.Error()})
		return
	}

	srcName := "main.tex"
	if engine == "typst" {
		srcName = "main.typ"
	}

	jobDir := filepath.Join(baseDir, "tmp", slug)

	_ = os.RemoveAll(jobDir)

	if err := os.MkdirAll(jobDir, 0750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	if err := os.WriteFile(filepath.Join(jobDir, srcName), []byte(document.Content), 0640); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	logOut, err := RunCompiler(engine, jobDir, srcName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":    false,
			"error": engine + " compilation failed",
			"log":   logOut,
		})
		utils.CleanupDirLater(jobDir, 5*time.Minute)
		return
	}

	pdfPath := filepath.Join(jobDir, "main.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"ok":    false,
			"error": "pdf not generated",
			"log":   logOut,
		})
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
		c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
		return
	}

	slug := c.Param("slug")
	if slug == "" {
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	if _, err := h.uc.GetBySlug(c.Request.Context(), authID, slug); err != nil {
		c.JSON(404, gin.H{"message": "Document not found: " + err.Error()})
		return
	}

	baseDir, err := utils.AppDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	pdfPath := filepath.Join(baseDir, "tmp", slug, "main.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "pdf not found"})
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
		c.JSON(400, gin.H{"message": "Slug is required"})
		return
	}

	_, err := h.uc.IsPublicBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(404, gin.H{"message": "Document not found: " + err.Error()})
		return
	}

	baseDir, err := utils.AppDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	pdfPath := filepath.Join(baseDir, "tmp", slug, "main.pdf")
	if _, err := os.Stat(pdfPath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "pdf not found"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `inline; filename="main.pdf"`)
	c.Header("Cache-Control", "no-store")
	c.File(pdfPath)
}
