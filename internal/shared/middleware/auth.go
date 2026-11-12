package middleware

import (
	"context"
	"time"
	"zelko/internal/shared/utils"

	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func AuthMiddleware(maker *utils.PasetoMaker, rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		authHeader := c.GetHeader("Authorization")
		if len(authHeader) < 7 || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		token := strings.TrimSpace(authHeader[7:])
		if token == "" || len(token) < 20 {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		payload, err := maker.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		c.Set("userID", payload.UserID)
		c.Set("payload", payload)
		c.Set("token", token)
		c.Next()
	}

}

func StorageAuthMiddleware(maker *utils.PasetoMaker, rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token := ""

		if len(authHeader) >= 7 && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		} else {
			cookieToken, err := c.Cookie("refresh_token")
			if err == nil && cookieToken != "" {
				token = cookieToken
			}
		}

		if token == "" || len(token) < 20 {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		payload, err := maker.VerifyToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"msg": "unauthorized"})
			return
		}

		c.Set("userID", payload.UserID)
		c.Set("payload", payload)
		c.Set("token", token)
		c.Next()
	}
}

func PreAuthMiddleware(rd *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("pre_auth_session")
		if err != nil || sessionID == "" {
			c.AbortWithStatusJSON(401, gin.H{"msg": "sesión de pre-autenticación requerida"})
			return
		}

		key := fmt.Sprintf("preauth:%s", sessionID)
		vals, err := rd.HGetAll(context.Background(), key).Result()
		if err != nil || len(vals) == 0 {
			c.AbortWithStatusJSON(401, gin.H{"msg": "sesión de pre-autenticación expirada o inválida"})
			return
		}

		authStage, ok := vals["auth_stage"]
		if !ok || authStage != "pending_2fa" {
			c.AbortWithStatusJSON(401, gin.H{"msg": "estado de autenticación inválido"})
			return
		}

		c.Set("pre_auth_session_id", sessionID)
		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = time.Now().UTC().Format("20060102150405")
		}
		c.Set("RequestID", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)
		c.Next()
	}
}
