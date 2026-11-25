package main

import (
	"context"
	"strings"
	"zelko/config"
	"zelko/config/db"
	"zelko/internal/routes"
	"zelko/internal/shared/helper"
	"zelko/internal/shared/middleware"

	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielkov/gin-helmet/ginhelmet"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Advertencia: no se encontró .env, se usan variables de entorno del sistema")
	}

	if err := db.Connect(); err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}

	if err := db.InitializeDatabase(); err != nil {
		fmt.Println("Error initializing database:", err)
		return
	}

	if err := config.InitRedis(context.Background()); err != nil {
		fmt.Println("Error initializing Redis:", err)
		return
	}

	HOST_URL_DEV := os.Getenv("HOST_URL_DEV")
	HOST_URL_PROD := os.Getenv("HOST_URL_PROD")
	HOST_URL_PROD_WWW := os.Getenv("HOST_URL_PROD_WWW")
	HOST_API_PROD := os.Getenv("HOST_API_PROD")
	HOST_API_PROD_WWW := os.Getenv("HOST_API_PROD_WWW")

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(gin.Logger(), middleware.SafeRecovery())
	r.HandleMethodNotAllowed = true
	r.NoRoute(middleware.NotFound)
	r.NoMethod(middleware.MethodNotAllowed)
	if strings.TrimSpace(os.Getenv("GO_ENV")) == "prod" {
		_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(ginhelmet.Default())
	r.Use(ginhelmet.Referrer())

	isProd := strings.TrimSpace(os.Getenv("GO_ENV")) == "prod"
	candidates := []string{HOST_URL_PROD, HOST_URL_PROD_WWW, HOST_API_PROD, HOST_API_PROD_WWW}
	if !isProd {
		candidates = append(candidates, HOST_URL_DEV)
	}

	allowOrigins := make([]string, 0, len(candidates))
	for _, o := range candidates {
		o = strings.TrimSpace(o)
		if o != "" && o != "*" {
			allowOrigins = append(allowOrigins, o)
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-Idempotency-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// solo se peude hacer 30 requests por segundo por IP, con un máximo de 30 requests en cola
	middleware.StartCleanup()
	r.Use(middleware.RateLimiterRedis(config.Rdb, "global", rate.Limit(5), 50))

	// Solo application/json y multipart/form-data, con límites de tamaño y validación de JSON
	r.MaxMultipartMemory = 8 << 20
	r.Use(middleware.RequestGuard(middleware.DefaultGuardConfig))

	// Los errores de validación usan los nombres del tag json (p. ej. first_name)
	helper.RegisterJSONTagName()

	routes.Init()

	api := r.Group("/api/v1")
	{
		routes.UserRoutes(api)
		routes.DocumentRoutes(api)
	}

	auth := api.Group("/auth")
	{
		routes.AuthRoutes(auth)
	}

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Welcome to the CV API",
		})
	})

	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "OK", "ip": c.ClientIP()})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	addr := ":4106"
	if strings.TrimSpace(os.Getenv("GO_ENV")) == "prod" {
		addr = "127.0.0.1:4106"
	}

	log.Println("Server starting on :4106...")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	if config.Rdb != nil {
		if err := config.Rdb.Close(); err != nil {
			log.Println("Error closing Redis:", err)
		}
	}

	db.CloseDB()

	log.Println("Server exiting")
}
