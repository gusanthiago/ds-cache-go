package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter creates a new router
func NewRouter(handler *Handler) *gin.Engine {
	service := gin.Default()

	service.GET("", func(context *gin.Context) {
		context.JSON(http.StatusOK, "Cache Manager")
	})

	service.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": "PAGE_NOT_FOUND"})
	})

	router := service.Group("/api/cache-manager/v1")

	router.GET("/", handler.GetByCacheKey)
	router.POST("/", handler.CreateCache)

	return service
}
