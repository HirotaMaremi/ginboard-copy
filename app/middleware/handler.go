package middleware

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"log"
	"os"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		err := c.Errors.ByType(gin.ErrorTypePublic).Last()
		if err != nil {
			log.Print(err.Err)

			env := os.Getenv("ENV")
			c.HTML(500, "error.html", gin.H{"msg": err.Err, "env": env})
			//c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			//	"Error": err.Error(),
			//})
		}
	}
}

func HeaderHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// キャッシュ設定
		//"no-cache, no-store, max-age=0, must-revalidate"
		c.Writer.Header().Set("Cache-Control", "max-age=0, private")

		// CORS設定
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		c.Next()
	}
}

func AdminAuthorize(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		loginUserId, _ := c.Get("userID")
		if !enforcer.HasNamedGroupingPolicy("g", fmt.Sprint(loginUserId), "admin") {
			env := os.Getenv("ENV")
			log.Print("admin権限がありません。")
			c.HTML(200, "error.html", gin.H{"msg": "不正なリクエストです。", "env": env})
			c.Abort()
			return
		}

		c.Next()
	}
}
