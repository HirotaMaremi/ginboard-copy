package utils

import (
	"github.com/gin-gonic/gin"
)

// https://qiita.com/CoGee/items/17ae184d62afb37dbe84
func SetFlash(c *gin.Context, status string, msg string) {
	c.SetCookie("Status", status, 1, "/", "localhost", true, true)
	c.SetCookie("Msg", msg, 1, "/", "localhost", true, true)
}
