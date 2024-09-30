package api

// TODO
// * min and max values should be configurable
// * create validate and better handlers

import (
	"ds-cache/api/helper"
	"ds-cache/api/response"
	"ds-cache/api/utils"
	"ds-cache/internal"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	NodesList []internal.DataNode
}

type CacheEntryRequest struct {
	Key   string `validate:"required,min=1,max=200" json:"key"`
	Value string `validate:"required,min=1,max=1000" json:"value"`
}

func NewHandler(nodeList []internal.DataNode) *Handler {
	return &Handler{NodesList: nodeList}
}

// CreateCache godoc
// @Summary Create a cache entry
// @Description Create a cache entry
// @Tags cache-manager
// @Accept  json
// @Produce  json
// @Param cacheEntry body CacheEntryRequest true "Cache Entry"
// @Success 200 {object} response.WebResponse
// @Router /api/cache-manager/v1 [post]
func (handler *Handler) CreateCache(ctx *gin.Context) {
	cacheEntry := CacheEntryRequest{}
	err := ctx.ShouldBindJSON(&cacheEntry)
	helper.ErrorPanic(err)

	currNode := utils.HashNode(cacheEntry.Key, len(handler.NodesList))
	go handler.NodesList[currNode].Put(cacheEntry.Key, cacheEntry.Value)

	fmt.Printf("Created Cache in node (%d) Key (%s) Value (%s) ", currNode, cacheEntry.Key, cacheEntry.Value)

	webResponse := response.WebResponse{
		Code:   200,
		Status: "Ok",
		Data:   cacheEntry,
	}

	ctx.JSON(http.StatusOK, webResponse)
}

// GetByCacheKey godoc
// @Summary Get a cache entry by key
// @Description Get a cache entry by key
// @Tags cache-manager
// @Accept  json
// @Produce  json
// @Param cacheKey path string true "Cache Key"
// @Success 200 {object} response.WebResponse
// @Router /api/cache-manager/v1/{cacheKey} [get]
func (handler *Handler) GetByCacheKey(ctx *gin.Context) {
	cacheKey := ctx.Query("cacheKey")

	currNode := utils.HashNode(cacheKey, len(handler.NodesList))
	value := handler.NodesList[currNode].Get(cacheKey)

	if value == "NOT_FOUND" {
		webResponse := response.WebResponse{
			Code:   404,
			Status: "Not Found",
			Data:   response.CacheResponse{Key: cacheKey, Value: "Not Found"},
		}

		ctx.JSON(http.StatusNotFound, webResponse)
		return
	}

	fmt.Printf("Found Cache in node (%d) Key (%s) Value (%s) ", currNode, cacheKey, value)

	webResponse := response.WebResponse{
		Code:   200,
		Status: "Ok",
		Data:   response.CacheResponse{Key: cacheKey, Value: value},
	}

	ctx.JSON(http.StatusOK, webResponse)
}
