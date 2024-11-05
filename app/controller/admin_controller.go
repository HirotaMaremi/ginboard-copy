package controller

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/service"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/rs/xid"
	csrf "github.com/utrack/gin-csrf"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// スレッド一覧画面
func AdminListThread(c *gin.Context) {
	//param := strings.TrimSpace(c.Query("param"))
	// adminでのtypeパラムは並び順ではなく、スレのtype
	params := c.Request.URL.Query()

	parseParams := map[string]string{}
	for key, val := range params {
		parseParams[key] = val[0]
	}
	//url := c.Request.URL.Path
	linkPath := c.Request.URL.RequestURI() + "?" // パラメータ付きURL
	if len(parseParams) != 0 {
		linkPath = c.Request.URL.RequestURI() + "&"
	}

	thCount, err := model.CountAdminThreads(parseParams)
	if err != nil {
		log.Println(err.Error())
		// adminページなので画面表示しても良い
		c.Error(err).SetType(gin.ErrorTypePublic)
	}
	page := utils.ConvertPage(c, int(thCount))
	threads, err2 := model.GetAdminThreadsPagination(page, parseParams)
	if err2 != nil {
		log.Println(err2.Error())
		c.Error(err2).SetType(gin.ErrorTypePublic)
	}

	// ページネーションの頁番号表示のための配列
	pages := make([]int, page.TotalPages, page.TotalPages)
	for i := 0; i < page.TotalPages; i++ {
		pages[i] = i + 1
	}
	Render(c, 200, "admin_list_thread.html", gin.H{
		"title":      "スレッド一覧",
		"threads":    threads,
		"pagination": page,
		"pages":      pages,
		"linkPath":   linkPath,
		"params":     parseParams,
	})
}

// スレッド作成・編集画面
func AdminShowCreateThread(c *gin.Context) {
	num := c.Param("threadNum")

	title := "スレッド作成"
	var thread model.Thread
	if num != "" {
		dbThread, err := model.GetOneThreadByNum(num)
		if err != nil {
			log.Println(err.Error())
			c.Error(err).SetType(gin.ErrorTypePublic)
			return
		}
		title = "スレッド編集"
		thread = dbThread
	}

	token := csrf.GetToken(c)
	Render(c, 200, "admin/create_thread.html", gin.H{
		"token":  token,
		"title":  title,
		"thread": thread,
	})
}

func AdminCreateThread(c *gin.Context) {
	token := csrf.GetToken(c)

	num := c.Param("threadNum")
	isValid, _ := strconv.ParseBool(c.PostForm("is_valid"))
	typeVal := c.PostForm("type")
	loginUserId, _ := c.Get("userID")
	detail := c.PostForm("detail")
	var thread model.Thread
	title := "スレッド作成"
	if num != "" {
		dbThread, err := model.GetOneThreadByNum(num)
		if err != nil {
			//errors.New("タイトルの入力は必須です。")
			c.Error(errors.New("該当のスレッドが存在しません。")).SetType(gin.ErrorTypePublic)
			return
		}
		title = "スレッド編集"
		thread = dbThread
	} else {
		thread.ThreadNumber = xid.New().String()
		thread.CreatedBy = uint(loginUserId.(float64)) //sessions.Default(c).Get("UserId").(uint)
	}
	thread.Detail = sql.NullString{detail, true}
	// statusを取得
	oldStatus := thread.Status

	if err := c.ShouldBind(&thread); err != nil {
		Render(c, 400, "admin/create_thread.html", gin.H{
			"token":  token,
			"title":  title,
			"status": "error",
			"msg":    err.Error(),
			"thread": thread,
		})
		return
	}
	thread.Type = sql.NullString{typeVal, true}
	thread.IsValid = isValid
	thread.Status, _ = strconv.Atoi(c.PostForm("status"))

	// 画像処理
	filename := service.ProcessImg(c)
	if filename != nil {
		thread.ImageFile = filename
	}

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
		}

		arrTagModel = append(arrTagModel, tag)
	}
	thread.Tags = arrTagModel

	validationErr := validateThread(thread)
	if validationErr != nil {
		Render(c, 400, "admin/create_thread.html", gin.H{
			"token":  token,
			"title":  title,
			"status": "error",
			"msg":    validationErr.Error(),
			"thread": thread,
		})
		return
	}

	if num != "" {
		model.DeleteRelationByThreadId(thread.ID)
		thread.UpdateThread()

		// 公開通知
		if oldStatus != thread.Status && thread.Status == 1 {
			threadUrl := ""
			switch thread.Type.String {
			case "normal":
				threadUrl = "/member/show/" + fmt.Sprint(thread.ThreadNumber)
			case "vote":
				threadUrl = "/member/vote/show/" + fmt.Sprint(thread.ThreadNumber)
			case "private":
				threadUrl = "/private/show/" + fmt.Sprint(thread.ThreadNumber)
			case "giin_private":
				threadUrl = "/private/giinShow/" + fmt.Sprint(thread.ThreadNumber)
			}
			var notice model.Notice
			notice.UserId = thread.CreatedBy
			notice.Title = "あなたが作成したスレッドが公開されました。"
			notice.Body = "公開されたスレッドは<br/><a href='" + threadUrl + "'>こちら</a>。"
			notice.IsValid = true
			notice.CreateNotice()
		}
	} else {
		thread.CreateThread()
	}

	c.Redirect(302, "/admin/thr")
}

// 通知検索
func AdminListNotice(c *gin.Context) {
	params := c.Request.URL.Query()

	parseParams := map[string]string{}
	for key, val := range params {
		parseParams[key] = val[0]
	}

	notices, err := model.GetAdminNoticeByUser(parseParams["user_id"])
	if err != nil {
		log.Println(err.Error())
		c.Error(err).SetType(gin.ErrorTypePublic)
	}

	Render(c, 200, "admin_list_notice.html", gin.H{
		"title":   "通知一覧",
		"notices": notices,
		"params":  parseParams,
	})
}

func AdminShowCreateNotice(c *gin.Context) {
	id := c.Param("id")

	title := "通知作成"
	var notice model.Notice
	if id != "" {
		dbNotice, err := model.GetNoticeById(id)
		if err != nil {
			log.Println(err.Error())
			c.Error(err).SetType(gin.ErrorTypePublic)
			return
		}
		title = "通知編集"
		notice = dbNotice
	}

	token := csrf.GetToken(c)
	Render(c, 200, "admin/create_notice.html", gin.H{
		"token":  token,
		"title":  title,
		"notice": notice,
	})
}

func AdminCreateNotice(c *gin.Context) {
	token := csrf.GetToken(c)

	id := c.Param("id")
	var notice model.Notice
	title := "通知作成"
	if id != "" {
		dbNotice, err := model.GetNoticeById(id)
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePublic)
			return
		}
		title = "通知編集"
		notice = dbNotice
	}
	if err := c.ShouldBind(&notice); err != nil {
		Render(c, 400, "admin/create_notice.html", gin.H{
			"token":  token,
			"title":  title,
			"status": "error",
			"msg":    err.Error(),
			"notice": notice,
		})
		return
	}

	validationErr := validateNotice(notice)
	if validationErr != nil {
		Render(c, 400, "admin/create_notice.html", gin.H{
			"token":  token,
			"title":  title,
			"status": "error",
			"msg":    validationErr.Error(),
			"notice": notice,
		})
		return
	}

	if id != "" {
		notice.UpdateNotice()
	} else {
		notice.CreateNotice()
	}

	c.Redirect(302, "/admin/notice?user_id="+fmt.Sprint(notice.UserId))
}

func validateNotice(notice model.Notice) error {
	_, err := model.GetOneWithErr(int(notice.UserId))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("存在しないユーザIDです。: " + fmt.Sprint(notice.UserId))
	}
	if len(notice.Title) == 0 {
		return errors.New("タイトルの入力は必須です。")
	}
	if len(notice.Body) == 0 {
		return errors.New("本文の入力は必須です。")
	}
	if len(notice.Title) > 100 {
		return errors.New("タイトルは100文字以内にしてください。")
	}
	if len(notice.Body) > 500 {
		return errors.New("本文は500文字未満で書いてください。")
	}
	return nil
}
