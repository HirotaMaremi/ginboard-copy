package controller

import (
	"database/sql"
	"errors"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/service"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	csrf "github.com/utrack/gin-csrf"
	"gorm.io/gorm"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func ShowTopThread(c *gin.Context) {
	url := c.Request.URL.Path
	strParam := c.Query("type")
	param := strings.TrimSpace(c.Query("param"))

	var fixThreads []model.Thread
	var errFix error
	fixThreads, errFix = model.GetFixThreads()
	if errFix != nil {
		log.Println(errFix.Error())
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	var data []model.Thread
	var err error
	isNew := false
	if strParam == "new" {
		isNew = true
		data, err = model.GetNewThreads("normal", param)
	} else {
		data, err = model.GetTopThreads("normal", param)
	}
	if err != nil {
		log.Println(err.Error())
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	// 未読通知があるか
	hasNotViewedNotice := false
	userId := utils.GetLoginUserId(c)
	if userId != 0 {
		notices, _ := model.GetNotViewedNoticeByUser(userId)
		if len(notices) != 0 {
			hasNotViewedNotice = true
		}
	}

	// CloudFront path
	cloudFrontDomain := os.Getenv("AWS_CLOUDFRONT_DOMAIN") + "/" + os.Getenv("ENV")

	template := "pc/index.html"
	if utils.IsSp(c) {
		template = "sp/index.html"
	}
	Render(c, 200, template, gin.H{
		"title":              "ginboard",
		"threads":            data,
		"fixThreads":         fixThreads,
		"url":                url,
		"isNew":              isNew,
		"imgDomain":          cloudFrontDomain,
		"hasNotViewedNotice": hasNotViewedNotice,
	})
}

// スレッド一覧画面
func ListThread(c *gin.Context) {
	ListThreadCommon("normal", "スレッド一覧", c)
}

// スレッド画面
func ShowOneThread(c *gin.Context) {
	num := c.Param("num")
	strCommentId := c.Query("comment")
	thread, err := model.GetOneThread(num, "normal")
	if err != nil {
		// fixスレッドも検索する
		fixThread, fixErr := model.GetOneThread(num, "fix")
		if fixErr != nil {
			log.Println(err.Error())
			c.Error(errors.New("スレッドが存在しません。")).SetType(gin.ErrorTypePublic)
			return
		}
		thread = fixThread
	}
	commentCount := len(thread.Comments)

	page := utils.ConvertPage(c, commentCount)
	comments, err := model.GetCommentPagination(page, thread.ID, strCommentId)
	likeCount, _ := model.GetLikePagination(page, thread.ID, strCommentId)
	if err != nil {
		log.Println(err.Error())
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	userMap := make(map[int]string)
	for _, val := range comments {
		userMap[int(val.ID)] = model.GetOne(int(val.CreatedBy)).Name
	}
	pages := make([]int, page.TotalPages, page.TotalPages)
	for i := 0; i < page.TotalPages; i++ {
		pages[i] = i + 1
	}

	// いいね機能
	var goodCommentIds []uint
	//var likeMap map[uint]gin.H
	likeMap := make(map[uint]gin.H)
	loginUserId, exist := c.Get("userID")
	if exist {
		idAndLikes, err := model.GetLikeCommentsByUserId(thread.ID, uint(loginUserId.(float64)))
		if err != nil {
			log.Println(err.Error())
			c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
			return
		}

		for _, v := range idAndLikes {
			// ログインユーザがいいねしたコメント一覧
			goodCommentIds = append(goodCommentIds, v.ID)
			// コメントごとのgood/badマッピング
			likeMap[v.ID] = gin.H{
				"Good": v.Good,
				"Bad":  v.Bad,
			}
		}
	}

	url := c.Request.URL.Path
	token := csrf.GetToken(c)
	imageUrl := ""
	if thread.ImageFile != nil {
		imageUrl = utils.GetImageUrl(thread.ImageFile)
	}

	hasDetail := false
	if thread.Detail.Valid == true && thread.Detail.String != "" {
		hasDetail = true
	}

	isMember := strings.HasPrefix(url, "/member/")

	template := "pc/show.html"
	if utils.IsSp(c) {
		template = "sp/show.html"
	}
	Render(c, 200, template, gin.H{
		"title":           thread.Title,
		"thread":          thread,
		"comments":        comments,
		"token":           token,
		"pagination":      page,
		"pages":           pages,
		"likeCount":       likeCount,
		"url":             url,
		"isMember":        isMember,
		"userMap":         userMap,
		"goodCommentIds":  goodCommentIds,
		"likeMap":         likeMap,
		"imageUrl":        imageUrl,
		"commentImageDir": utils.GetCommentImageDir(thread.ID),
		"hasDetail":       hasDetail,
	})
}

// スレッド作成・編集画面
func ShowCreateThread(c *gin.Context) {
	num := c.Param("threadNum")

	isEdit := false
	title := "スレッド作成"
	var thread model.Thread
	if num != "" {
		dbThread, err := model.GetOneThreadByNum(num)
		if err != nil {
			log.Println(err.Error())
			c.Error(errors.New("該当のスレッドが存在しません。")).SetType(gin.ErrorTypePublic)
			return
		}
		isEdit = true
		title = "スレッド編集"
		thread = dbThread
	}

	token := csrf.GetToken(c)
	template := "pc/create_thread.html"
	if utils.IsSp(c) {
		template = "sp/create_thread.html"
	}
	Render(c, 200, template, gin.H{
		"token":  token,
		"title":  title,
		"thread": thread,
		"isEdit": isEdit,
	})
}

func CreateThread(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := csrf.GetToken(c)
		num := c.Param("threadNum")
		loginUserId, _ := c.Get("userID")
		detail := c.PostForm("detail")
		var thread model.Thread
		isEdit := false
		title := "スレッド作成"
		if num != "" {
			dbThread, err := model.GetOneThreadByNum(num)
			if err != nil {
				c.Error(errors.New("該当のスレッドが存在しません。")).SetType(gin.ErrorTypePublic)
				return
			}
			isEdit = true
			title = "スレッド編集"
			thread = dbThread
		} else {
			thread.Type = sql.NullString{"normal", true}
			thread.ThreadNumber = xid.New().String()
			thread.CreatedBy = uint(loginUserId.(float64)) //sessions.Default(c).Get("UserId").(uint)
		}
		thread.Detail = sql.NullString{detail, true}

		template := "pc/create_thread.html"
		if utils.IsSp(c) {
			template = "sp/create_thread.html"
		}
		// https://gin-gonic.com/ja/docs/examples/binding-and-validation/
		if err := c.ShouldBind(&thread); err != nil {
			Render(c, 400, template, gin.H{
				"token":  token,
				"title":  title,
				"status": "error",
				"msg":    err.Error(),
				"thread": thread,
				"isEdit": isEdit,
			})
			return
		}

		// 画像処理
		filename := service.ProcessImg(c)
		if filename != nil {
			thread.ImageFile = filename
		}

		//タグ系処理
		tags := c.Request.Form.Get("tags")
		arrTags := strings.Split(tags, ",")
		var arrTagModel []model.Tag
		for _, v := range arrTags {
			if v == "" {
				continue
			}
			var tag model.Tag
			tag, err := model.GetOneTag(v)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tag.Name = v
				//c.Error(errors.New("some message"))
			}

			arrTagModel = append(arrTagModel, tag)
		}
		thread.Tags = arrTagModel

		validationErr := validateThread(thread)
		if validationErr != nil {
			Render(c, 400, template, gin.H{
				"token":  token,
				"title":  title,
				"status": "error",
				"msg":    validationErr.Error(),
				"thread": thread,
				"isEdit": isEdit,
			})
			return
		}

		if num != "" {
			model.DeleteRelationByThreadId(thread.ID)
			thread.UpdateThread()
		} else {
			// status=0で生成されるのですぐに公開されない。
			thread.CreateThread()
		}
		// ログ登録
		var log model.Log
		log.UserId = uint(loginUserId.(float64))
		log.Action = "create thread"
		log.IpAddress = c.ClientIP()
		log.CreateLog()

		session := sessions.Default(c)
		session.AddFlash("スレッドを作成しました。公開まで今しばらくお待ちください。")
		session.Save()

		c.Redirect(302, "/member")
	}
}

func PostImageComment(c *gin.Context) {
	//token := csrf.GetToken(c)
	id, _ := strconv.Atoi(c.PostForm("thread_id"))
	loginUserId, _ := c.Get("userID")
	var comment model.Comment
	//title := "スレッド作成"
	dbThread, err := model.GetOneThreadById(uint(id))
	if err != nil {
		c.Error(errors.New("該当のスレッドが存在しません。")).SetType(gin.ErrorTypePublic)
		return
	}

	// 画像処理
	filename := service.ProcessCommentImg(c, uint(id))
	if filename != nil {
		file := *filename
		comment.Comment = file
	} else {
		c.Error(errors.New("画像処理に失敗しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	lastThread := model.GetLastComment(dbThread.ID)
	comment.CommentNumber = lastThread.CommentNumber + 1
	comment.CreatedBy = uint(loginUserId.(float64))
	comment.ThreadId = dbThread.ID
	comment.IsValid = true
	comment.Type = 2
	comment.CreateComment()

	// ユーザのLastCommentedAt更新
	loginUser := model.GetOne(int(loginUserId.(float64)))
	now := time.Now()
	loginUser.LastCommentedAt = &now
	loginUser.Update()

	// スレッドのLastCommentedAt更新
	thread, _ := model.GetOneThreadById(comment.ThreadId)
	thread.LastCommentedAt = &now
	thread.UpdateThread()

	// ログ登録
	var log model.Log
	log.UserId = loginUser.ID
	log.Action = "post image comment"
	log.IpAddress = c.ClientIP()
	log.CreateLog()

	url := "/member/show/"
	if dbThread.Type.String == "private" {
		url = "/private/show/"
	}
	if dbThread.Type.String == "vote" {
		url = "/member/vote/show/"
	}
	if dbThread.Type.String == "giin_private" {
		url = "/private/giinShow/"
	}
	c.Redirect(302, url+dbThread.ThreadNumber)
}

func validateThread(thread model.Thread) error {
	if len(thread.Title) == 0 {
		return errors.New("タイトルの入力は必須です。")
	}
	if len(thread.Description) == 0 {
		return errors.New("説明の入力は必須です。")
	}
	if len([]rune(thread.Title)) > 200 {
		return errors.New("タイトルは200文字以内にしてください。")
	}
	if len([]rune(thread.Description)) > 500 {
		return errors.New("説明は500文字未満で書いてください。")
	}
	detail := thread.Detail.String
	if thread.Detail.Valid != false && len([]rune(detail)) > 100000 {
		return errors.New("詳細は100000文字未満で書いてください。")
	}
	return nil
}
