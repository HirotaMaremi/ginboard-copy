package middleware

import (
	"fmt"
	"github.com/HirotaMaremi/ginboard/utils"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// AuthorizeJWT -> to authorize JWT Token
func AuthorizeJWT() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString, err := ctx.Cookie("token")
		if err != nil {
			log.Println(err.Error())
			ctx.Redirect(302, "/signin")
			ctx.Abort()
			return
		}

		if token, err := utils.ValidateToken(tokenString); err != nil {

			fmt.Println("token", tokenString, err.Error())
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Not Valid Token"})

		} else {

			if claims, ok := token.Claims.(jwt.MapClaims); !ok {
				ctx.AbortWithStatus(http.StatusUnauthorized)

			} else {
				if token.Valid {
					ctx.Set("userID", claims["userID"])
					userIDStr := claims["userID"].(float64)
					newToken := utils.GenerateToken(uint(userIDStr))
					utils.SetCookie(ctx, "token", newToken)
					ctx.Next()
				} else {
					ctx.AbortWithStatus(http.StatusUnauthorized)
				}

			}
		}

	}

}

// https://zurustar.hatenablog.com/entry/2022/03/27/160948
func AuthorizeUser(c *gin.Context) {
	// Authorizationヘッダからトークンを取り出す
	authorizationHeader := c.Request.Header.Get("Authorization")
	if authorizationHeader != "" {
		ary := strings.Split(authorizationHeader, " ")
		if len(ary) == 2 {
			if ary[0] == "Bearer" {
				// ここまででトークンと思われるものを取り出せたので、解析する
				// ここのロジックは公式サイト参照
				//　https://pkg.go.dev/github.com/golang-jwt/jwt#example-Parse-Hmac
				token, err := jwt.Parse(ary[1], func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
					}
					return []byte(os.Getenv("JWT_SECRET")), nil
				})
				if err == nil {
					// 解析に成功したら、ユーザIDを取り出す
					if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
						userIDStr := claims["sub"].(string)
						userID, err := strconv.Atoi(userIDStr)
						if err == nil {
							// ユーザIDを使って新しいトークンを取り出す（有効期限切れ対策）
							newToken := utils.GenerateToken(uint(userID))
							c.Set("token", newToken)
							c.Set("userID", userID)
						}
					}
				}
			}
		}
	}
	c.Next()
}
