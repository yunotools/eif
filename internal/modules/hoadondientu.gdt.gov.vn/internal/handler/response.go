package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/yunotools/eif/internal/core/apperr"
)

const sessionHeader = "X-Session-ID"

func writeError(c *gin.Context, err error) {
	res := apperr.FromError(err, c.GetString("request_id"))
	c.JSON(res.StatusCode, res)
}

func sessionID(c *gin.Context) string {
	return c.GetHeader(sessionHeader)
}
