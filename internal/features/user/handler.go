package user

import (
	"log"
	"net/http"
	"zelko/internal/shared/helper"
	"zelko/internal/shared/utils"

	"github.com/gin-gonic/gin"
)

type CreateUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type UpdateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
}

type UserHandler struct {
	uc Service
}

func NewUserHandler(uc Service) *UserHandler {
	return &UserHandler{uc: uc}
}

func (h *UserHandler) GetByAuthID(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	user, err := h.uc.GetByAuthID(c.Request.Context(), authID)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": user, "message": "User retrieved successfully"})
}

func (h *UserHandler) Create(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		log.Printf("Error extracting authID: %v", err)
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	log.Printf("Creating user with authID: %d", authID)

	var user CreateUserRequest
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(400, gin.H{"message": "Invalid request body"})
		return
	}

	_, err = h.uc.Create(c.Request.Context(), authID, user.FirstName, user.LastName)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(500, gin.H{"message": "Error creating user: " + err.Error()})
		return
	}

	c.JSON(201, gin.H{"message": "User created successfully", "data": user})
}

func (h *UserHandler) Update(c *gin.Context) {
	_, authID, err := utils.ExtractedParamsJwt(c)
	if err != nil {
		c.JSON(400, gin.H{"message": "Invalid ID: " + err.Error()})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "Datos inválidos",
			"errors":  helper.HumanValidationErrors(err),
		})
		return
	}

	err = h.uc.Update(c.Request.Context(), authID, req.FirstName, req.LastName)
	if err != nil {
		c.JSON(500, gin.H{"message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "User updated successfully"})
}

func (h *UserHandler) UploadAvatar(c *gin.Context) {
	// _, authID, err := utils.ExtractedParamsJwt(c)
	// if err != nil {
	// 	c.JSON(400, gin.H{"message": "Invalid auth: " + err.Error()})
	// 	return
	// }

	// file, size, mimeType, err := utils.ExtractImage(c, "image")
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	// 	return
	// }

	// fileId, path, err := utils.SaveFile("users/avatars", "", file, c)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving avatar: " + err.Error()})
	// 	return
	// }

	// err = h.uc.UploadAvatar(c.Request.Context(), authID, fileId, path, mimeType, size)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{"message": "Avatar uploaded successfully"})
}
