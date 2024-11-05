package utils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"strings"
)

func SetCookie(ctx *gin.Context, key string, value string) {
	// MaxAge3時間
	ctx.SetCookie(key, value, 3600*3*1, "", "", true, true)
}

func DeleteCookie(ctx *gin.Context, key string) {
	ctx.SetCookie(key, "", -1, "/", "", false, true)
}

func IsSp(ctx *gin.Context) bool {
	ua := ctx.GetHeader("User-Agent")

	if strings.Contains(ua, "iPhone") ||
		strings.Contains(ua, "iPod") ||
		strings.Contains(ua, "Android") {
		return true
	}
	// iPadはPC表示
	return false
}

func GetRootUrl(ctx *gin.Context) string {
	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}

	return scheme + "://" + ctx.Request.Host
}

func GetUrl(ctx *gin.Context) string {
	scheme := "http"
	if ctx.Request.TLS != nil {
		scheme = "https"
	}

	return scheme + "://" + ctx.Request.Host + ctx.Request.URL.Path
}

func MapMerge(m ...map[string]interface{}) map[string]interface{} {
	ans := make(map[string]interface{}, 0)

	for _, c := range m {
		for k, v := range c {
			ans[k] = v
		}
	}
	return ans
}

func StrContains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func RemoveStrArrayDuplicate(args []string) []string {
	results := make([]string, 0, len(args))
	encountered := map[string]bool{}
	for i := 0; i < len(args); i++ {
		if !encountered[args[i]] {
			encountered[args[i]] = true
			results = append(results, args[i])
		}
	}
	return results
}

func GetLoginUserId(c *gin.Context) uint {
	loginUserId, _ := c.Get("userID")
	if loginUserId == nil {
		return 0
	}
	id := uint(loginUserId.(float64))
	return id
}

func CreateFolder(dirname string) error {
	_, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(dirname, 0755)
		if errDir != nil {
			return errDir
		}
	}
	return nil
}

func GetImageUrl(fileName *string) string {
	env := os.Getenv("ENV")
	cloudFrontDomain := os.Getenv("AWS_CLOUDFRONT_DOMAIN")
	imageUrl := cloudFrontDomain + "/" + env + "/" + *fileName

	return imageUrl
}

func GetCommentImageDir(threadId uint) string {
	env := os.Getenv("ENV")
	cloudFrontDomain := os.Getenv("AWS_CLOUDFRONT_DOMAIN")
	imageDir := cloudFrontDomain + "/" + env + "/comment/" + fmt.Sprint(threadId)

	return imageDir
}
