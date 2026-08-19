package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type WebResponse struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
}

type CacheResponse struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// JSON writes the given data inside the response envelope with the given code.
func JSON(ctx *gin.Context, code int, data any) {
	ctx.JSON(code, WebResponse{
		Code:   code,
		Status: http.StatusText(code),
		Data:   data,
	})
}

// Fail writes the given message inside the response envelope with the given
// code, and aborts the chain so no later handler writes a second body.
func Fail(ctx *gin.Context, code int, message string) {
	ctx.AbortWithStatusJSON(code, WebResponse{
		Code:   code,
		Status: http.StatusText(code),
		Error:  message,
	})
}
