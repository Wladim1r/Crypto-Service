package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getFromCtx[T any](c *gin.Context, key string) (T, bool) {
	var zeroValue T

	anyVal, exists := c.Get(key)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("context var %s does not exist", key),
		})
		return zeroValue, false
	}

	value := anyVal.(T)

	return value, true
}
