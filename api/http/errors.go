package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrNotFound(c *gin.Context, err error) {
	Err(c, http.StatusNotFound, err)
}

func ErrBadRequest(c *gin.Context, err error) {
	Err(c, http.StatusBadRequest, err)
}

func ErrInternalServerError(c *gin.Context, err error) {
	Err(c, http.StatusInternalServerError, err)
}

func Err(c *gin.Context, code int, err error) {
	c.JSON(code, gin.H{"error": err.Error()})
}
