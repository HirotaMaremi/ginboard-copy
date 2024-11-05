package controller

import (
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"strings"
)

func Render(c *gin.Context, status int, template string, val gin.H) {
	//_, exist := c.Get("userID")
	_, err := c.Cookie("token")
	exist := err == nil
	url := c.Request.URL.Path //相対パス(スラッシュ始まり)
	isMember := strings.HasPrefix(url, "/member/")
	isPrivate := strings.HasPrefix(url, "/private/")
	isAccount := strings.HasPrefix(url, "/account/")
	isAdmin := strings.HasPrefix(url, "/admin/")
	session := sessions.Default(c)
	flashes := session.Flashes()
	session.Save()

	env := os.Getenv("ENV")
	host := c.Request.Host

	mustValue := gin.H{
		"isLogin":   exist,
		"isMember":  isMember,
		"isPrivate": isPrivate,
		"isAccount": isAccount,
		"isAdmin":   isAdmin,
		"flashes":   flashes,
		"env":       env,
		"host":      host,
	}
	param := utils.MapMerge(mustValue, val)

	c.HTML(status, template, param)
}

// スレッド種別共通の一覧表示画面
func ListThreadCommon(thrType string, title string, c *gin.Context) {
	param := strings.TrimSpace(c.Query("param"))
	orderParam := c.Query("type")
	tagParam := c.Query("tag")
	url := c.Request.URL.Path
	//requestUrl := c.Request.URL.RequestURI() // パラメータ付きURL
	linkPath := url + "?type=" + orderParam + "&param=" + param + "&"

	isNew := false
	if orderParam == "new" {
		isNew = true
	}

	thCount, err := model.CountThreadsByTypeParam(thrType, param, tagParam)
	if err != nil {
		log.Println(err.Error())
	}
	page := utils.ConvertPage(c, int(thCount))
	threads, err2 := model.GetThreadsPaginationByTypeOrderParam(page, thrType, orderParam, param, tagParam)
	if err2 != nil {
		log.Println(err2.Error())
	}

	// ページネーションの頁番号表示のための配列
	pages := make([]int, page.TotalPages, page.TotalPages)
	for i := 0; i < page.TotalPages; i++ {
		pages[i] = i + 1
	}

	// CloudFront path
	cloudFrontDomain := os.Getenv("AWS_CLOUDFRONT_DOMAIN") + "/" + os.Getenv("ENV")

	isGiin := false
	var giinThreads []model.Thread
	// giin_private_list_thread
	template := "pc/list_thread.html"
	if thrType == "private" {
		giinThreads, _ = model.GetGiinPrivateThread()
		template = "pc/private_list_thread.html"
		if utils.IsSp(c) {
			template = "sp/private_list_thread.html"
		}
	} else if thrType == "normal" {
		template = "pc/list_thread.html"
		if utils.IsSp(c) {
			template = "sp/list_thread.html"
		}
	}
	Render(c, 200, template, gin.H{
		"title":       title,
		"threads":     threads,
		"pagination":  page,
		"pages":       pages,
		"url":         url,
		"isNew":       isNew,
		"linkPath":    linkPath,
		"param":       param,
		"imgDomain":   cloudFrontDomain,
		"isGiin":      isGiin,
		"giinThreads": giinThreads,
	})
}
