package api

import (
	"errors"
	"net/http"

	"ds-cache/api/response"
	"ds-cache/internal/service"

	"github.com/gin-gonic/gin"
)

const keyParam = "cacheKey"

type Handler struct {
	cache *service.CacheService
}

// CacheEntryRequest is the body of a write. The limits use `binding` tags,
// which is what the gin validator reads.
type CacheEntryRequest struct {
	Key   string `json:"key" binding:"required,min=1,max=200"`
	Value string `json:"value" binding:"required,min=1,max=1000"`
}

// NewHandler creates a new Handler over the given cache service.
// Returns a pointer to the new Handler.
func NewHandler(cache *service.CacheService) *Handler {
	return &Handler{cache: cache}
}

// CreateCache godoc
// @Summary Create or update a cache entry
// @Description Stores a value under a key on the node that owns it
// @Tags cache-manager
// @Accept  json
// @Produce  json
// @Param cacheEntry body CacheEntryRequest true "Cache Entry"
// @Success 200 {object} response.WebResponse "Existing key updated"
// @Success 201 {object} response.WebResponse "New key created"
// @Failure 400 {object} response.WebResponse
// @Router /api/cache-manager/v1 [post]
func (h *Handler) CreateCache(ctx *gin.Context) {
	var entry CacheEntryRequest
	if err := ctx.ShouldBindJSON(&entry); err != nil {
		response.Fail(ctx, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.cache.Set(entry.Key, entry.Value)
	if err != nil {
		h.fail(ctx, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}

	response.JSON(ctx, status, response.CacheResponse{Key: entry.Key, Value: entry.Value})
}

// GetByCacheKey godoc
// @Summary Get a cache entry by key
// @Description Reads a key from the node that owns it
// @Tags cache-manager
// @Produce  json
// @Param cacheKey query string true "Cache Key"
// @Success 200 {object} response.WebResponse
// @Failure 400 {object} response.WebResponse
// @Failure 404 {object} response.WebResponse
// @Router /api/cache-manager/v1 [get]
func (h *Handler) GetByCacheKey(ctx *gin.Context) {
	key := ctx.Query(keyParam)

	value, found, err := h.cache.Get(key)
	if err != nil {
		h.fail(ctx, err)
		return
	}
	if !found {
		response.Fail(ctx, http.StatusNotFound, "cache key not found")
		return
	}

	response.JSON(ctx, http.StatusOK, response.CacheResponse{Key: key, Value: value})
}

// DeleteByCacheKey godoc
// @Summary Delete a cache entry by key
// @Description Removes a key from every layer of the node that owns it
// @Tags cache-manager
// @Produce  json
// @Param cacheKey query string true "Cache Key"
// @Success 200 {object} response.WebResponse
// @Failure 400 {object} response.WebResponse
// @Failure 404 {object} response.WebResponse
// @Router /api/cache-manager/v1 [delete]
func (h *Handler) DeleteByCacheKey(ctx *gin.Context) {
	key := ctx.Query(keyParam)

	removed, err := h.cache.Delete(key)
	if err != nil {
		h.fail(ctx, err)
		return
	}
	if !removed {
		response.Fail(ctx, http.StatusNotFound, "cache key not found")
		return
	}

	response.JSON(ctx, http.StatusOK, response.CacheResponse{Key: key})
}

// Stats godoc
// @Summary Cluster cache statistics
// @Description Per-node, per-layer hit, miss and eviction counters
// @Tags cache-manager
// @Produce  json
// @Success 200 {object} response.WebResponse
// @Router /api/cache-manager/v1/stats [get]
func (h *Handler) Stats(ctx *gin.Context) {
	response.JSON(ctx, http.StatusOK, h.cache.Stats())
}

// fail writes the given service error with the status code that matches it.
func (h *Handler) fail(ctx *gin.Context, err error) {
	if errors.Is(err, service.ErrEmptyKey) {
		response.Fail(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Fail(ctx, http.StatusInternalServerError, err.Error())
}
