package api

import (
	"log/slog"
	"net/http"
	"time"

	"ds-cache/api/response"

	"github.com/gin-gonic/gin"
)

const basePath = "/api/cache-manager/v1"

// NewRouter creates a new router for the given handler.
func NewRouter(handler *Handler) *gin.Engine {
	service := gin.New()
	service.Use(requestLogger(), gin.Recovery())

	service.GET("/", func(ctx *gin.Context) {
		response.JSON(ctx, http.StatusOK, gin.H{"service": "cache-manager"})
	})

	service.GET("/health", func(ctx *gin.Context) {
		response.JSON(ctx, http.StatusOK, gin.H{"status": "ok"})
	})

	service.NoRoute(func(ctx *gin.Context) {
		response.Fail(ctx, http.StatusNotFound, "PAGE_NOT_FOUND")
	})

	router := service.Group(basePath)

	// Both path forms are registered so a POST is never redirected, which
	// would drop its body for clients that do not follow.
	for _, path := range []string{"", "/"} {
		router.GET(path, handler.GetByCacheKey)
		router.POST(path, handler.CreateCache)
		router.DELETE(path, handler.DeleteByCacheKey)
	}

	router.GET("/stats", handler.Stats)

	return service
}

// requestLogger returns a middleware that logs one line per request through
// slog, replacing the gin logger.
func requestLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		slog.Info("request",
			"method", ctx.Request.Method,
			"path", ctx.Request.URL.Path,
			"status", ctx.Writer.Status(),
			"duration", time.Since(start),
		)
	}
}
