package controller

import (
	"errors"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/service"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
	"log"
)

func ShowPrivateThread(c *gin.Context) {
	ListThreadCommon("private", "Privateスレッド一覧", c)
}

// スレッド画面
func ShowOnePrivateThread(c *gin.Context) {
	num := c.Param("num")
	thread, err := model.GetOneThread(num, "private")
	if err != nil {
		log.Println(err.Error())
		c.Error(errors.New("スレッドが存在しません。")).SetType(gin.ErrorTypePublic)
		return
	}

	viewModel, err := service.CreateThreadViewModel(thread, c)
	if err != nil {
		c.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	imageUrl := ""
	if thread.ImageFile != nil {
		imageUrl = utils.GetImageUrl(thread.ImageFile)
	}
	hasDetail := false
	if thread.Detail.Valid == true && thread.Detail.String != "" {
		hasDetail = true
	}

	url := c.Request.URL.Path
	token := csrf.GetToken(c)

	template := "pc/private_show.html"
	if utils.IsSp(c) {
		template = "sp/private_show.html"
	}
	Render(c, 200, template, gin.H{
		"title":           thread.Title,
		"thread":          thread,
		"comments":        viewModel.Comments,
		"token":           token,
		"pagination":      viewModel.Pagination,
		"pages":           viewModel.Pages,
		"likeCount":       viewModel.LikeCount,
		"url":             url,
		"userMap":         viewModel.UserMap,
		"goodCommentIds":  viewModel.GoodCommentIds,
		"likeMap":         viewModel.LikeMap,
		"imageUrl":        imageUrl,
		"commentImageDir": utils.GetCommentImageDir(thread.ID),
		"hasDetail":       hasDetail,
	})
}

// 2023年選出議員のスレ
func ShowOneGiinThread(c *gin.Context) {
	num := c.Param("num")
	thread, err := model.GetOneThread(num, "giin_private")
	if err != nil {
		log.Println(err.Error())
		c.Error(errors.New("スレッドが存在しません。")).SetType(gin.ErrorTypePublic)
		return
	}

	viewModel, err := service.CreateThreadViewModel(thread, c)
	if err != nil {
		c.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	imageUrl := ""
	if thread.ImageFile != nil {
		imageUrl = utils.GetImageUrl(thread.ImageFile)
	}
	hasDetail := false
	if thread.Detail.Valid == true && thread.Detail.String != "" {
		hasDetail = true
	}

	partys, err2 := model.GetAllNestedParty()
	if err2 != nil {
		c.Error(err2).SetType(gin.ErrorTypePublic)
		return
	}
	// 無所属議員の取得
	noPartyGiins, err3 := model.GetNoPartyGiins("2023")
	if err3 != nil {
		c.Error(err3).SetType(gin.ErrorTypePublic)
		return
	}

	loginUserIdTmp, _ := c.Get("userID")
	loginUserId := uint(loginUserIdTmp.(float64))

	url := c.Request.URL.Path
	token := csrf.GetToken(c)

	template := "pc/private_show.html"
	if utils.IsSp(c) {
		template = "sp/private_show.html"
	}

	Render(c, 200, template, gin.H{
		"title":           thread.Title,
		"thread":          thread,
		"comments":        viewModel.Comments,
		"token":           token,
		"pagination":      viewModel.Pagination,
		"pages":           viewModel.Pages,
		"likeCount":       viewModel.LikeCount,
		"url":             url,
		"userMap":         viewModel.UserMap,
		"goodCommentIds":  viewModel.GoodCommentIds,
		"likeMap":         viewModel.LikeMap,
		"imageUrl":        imageUrl,
		"commentImageDir": utils.GetCommentImageDir(thread.ID),
		"hasDetail":       hasDetail,
		"partyGiins":      partys,
		"noPartyGiins":    noPartyGiins,
		"isGiin":          true,
		"loginUserId":     loginUserId,
	})
}
