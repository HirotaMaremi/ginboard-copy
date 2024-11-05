package controller

import (
	"errors"
	"fmt"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
	"gorm.io/gorm"
	"log"
	"math/rand"
	"regexp"
	"time"
)

// アカウントTOP
func AccountTop(c *gin.Context) {
	loginUserId, _ := c.Get("userID")
	user := model.GetActiveOne(int(loginUserId.(float64)))
	token := csrf.GetToken(c)
	template := "pc/account_top.html"
	if utils.IsSp(c) {
		template = "sp/account_top.html"
	}
	Render(c, 200, template, gin.H{
		"user":  user,
		"token": token,
		"title": "登録情報",
	})
}

// 　パスワード変更
func ChangePassword(c *gin.Context) {
	loginUserId, _ := c.Get("userID")
	user := model.GetActiveOne(int(loginUserId.(float64)))
	token := csrf.GetToken(c)

	oldPass := c.PostForm("old-pass")
	if isTrue := utils.ComparePassword(user.Password, oldPass); !isTrue {
		c.JSON(200, gin.H{
			"success": false,
			"token":   token,
			"message": "パスワードが不正です。",
		})
		return
	}
	// 入力された２件のパスワードの一致確認
	pass := c.PostForm("password")
	pass2 := c.PostForm("password2")
	if pass != pass2 {
		c.JSON(200, gin.H{
			"success": false,
			"token":   token,
			"message": "パスワードが一致していません。",
		})
		return
	}

	validationErr := validatePassword(pass)
	if validationErr != nil {
		c.JSON(200, gin.H{
			"success": false,
			"token":   token,
			"message": validationErr.Error(),
		})
		return
	}
	utils.HashPassword(&pass)
	user.Password = pass
	user.Update()

	c.JSON(200, gin.H{
		"success": true,
		"message": "パスワードを変更しました。",
	})
}

// 　メールアドレス変更
func ChangeEmail(c *gin.Context) {
	loginUserId, _ := c.Get("userID")
	user := model.GetActiveOne(int(loginUserId.(float64)))
	token := csrf.GetToken(c)

	oldEmail := user.Email
	newEmail := c.PostForm("new-email")
	if oldEmail == newEmail {
		c.JSON(200, gin.H{
			"success": false,
			"token":   token,
			"message": "変更前のメールアドレスと同一です。",
		})
		return
	}
	user.Email = newEmail
	user.Update()
	// 確認メール送信
	utils.SendEmailChangeMail(user.Email)

	// ログ登録
	var log model.Log
	log.UserId = user.ID
	log.Action = "change email. old address: " + oldEmail
	log.IpAddress = c.ClientIP()
	log.CreateLog()

	c.JSON(200, gin.H{
		"success": true,
		"message": "メールアドレスを変更しました。<br>変更後のメールアドレスに確認のためのメールが届いていることを確認してください。<br>届いていない場合、管理人まで問い合わせてください。",
	})
}

// 紹介コード画面（生成機能と既存コードの一覧画面）
func ShowCode(c *gin.Context) {
	// 既存コード取得
	loginUserId, _ := c.Get("userID")
	// uint(loginUserId.(float64))
	codes, _ := model.FindInviteCodesByUser(uint(loginUserId.(float64)))
	n := time.Now()
	//https://pkg.go.dev/github.com/withgame/gin-csrf
	token := csrf.GetToken(c)
	template := "pc/codes.html"
	if utils.IsSp(c) {
		template = "sp/codes.html"
	}
	Render(c, 200, template, gin.H{
		"codes": codes,
		"token": token,
		"now":   n,
		"title": "紹介コード",
	})
}

// ajax API 紹介コード生成
func CreateCode(c *gin.Context) {
	var runes = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	//紹介コードは８文字のランダム文字列とする
	b := make([]rune, 8)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	loginUserId, _ := c.Get("userID")

	// 1日の発行限度数 = 30
	count, _ := model.CountTodayInviteCode(uint(loginUserId.(float64)))
	if count >= 30 {
		c.JSON(200, gin.H{
			"success": false,
			"message": "発行限度数に達しています。",
		})
		return
	}

	var inviteCode model.InviteCode
	for i := 0; i < 10; i++ {
		code, err := model.FindInviteCode(string(b))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			code.Code = string(b)
			code.UserId = uint(loginUserId.(float64))
			// 有効期限は２週間
			code.ExpireAt = time.Now().AddDate(0, 0, 14)
			code.CreateInviteCode()

			inviteCode = code
			break
		}
		inviteCode.Code = ""
	}

	if inviteCode.Code == "" {
		c.JSON(200, gin.H{
			"success": false,
			"message": "エラーが発生しました。",
		})
		log.Println("コード生成失敗")
	} else {
		c.JSON(200, gin.H{
			"success": true,
			"code":    inviteCode.Code,
		})
	}
}

// 作成したスレッド一覧
func ListAccountThreads(c *gin.Context) {
	loginUserId, _ := c.Get("userID")
	threads, _ := model.GetAllThreadByUser(uint(loginUserId.(float64)))

	template := "pc/account_threads.html"
	if utils.IsSp(c) {
		template = "sp/account_threads.html"
	}
	Render(c, 200, template, gin.H{
		"title":   "作成済スレッド",
		"threads": threads,
	})
}

// 通知一覧
func ListNotices(c *gin.Context) {
	loginUserId, _ := c.Get("userID")
	notices, _ := model.GetNoticeByUser(uint(loginUserId.(float64)))
	token := csrf.GetToken(c)
	template := "pc/account_notices.html"
	if utils.IsSp(c) {
		template = "sp/account_notices.html"
	}
	Render(c, 200, template, gin.H{
		"title":   "通知一覧",
		"notices": notices,
		"token":   token,
	})
}

// AJAX api
func ViewedNotice(c *gin.Context) {
	noticeId := c.PostForm("id")
	notice, err := model.GetNoticeById(noticeId)
	if err != nil {
		log.Println(err.Error())
		c.JSON(200, gin.H{
			"success": false,
		})
		return
	}
	notice.IsViewed = true
	notice.UpdateNotice()
	c.JSON(200, gin.H{
		"success": true,
	})
}

// 退会処理
func Withdrawal(c *gin.Context) {
	token := csrf.GetToken(c)
	template := "pc/withdrawal.html"
	if utils.IsSp(c) {
		template = "sp/withdrawal.html"
	}
	Render(c, 200, template, gin.H{
		"title": "退会",
		"token": token,
	})
}

func ExecuteWithdrawal(c *gin.Context) {
	loginUserId, _ := c.Get("userID")
	loginUser := model.GetOne(int(loginUserId.(float64)))
	oldName := loginUser.Name
	// 退会処理ではis_active=false, ユーザ名を"anonymous", emailを適当な文字列に書き換える
	loginUser.Name = "anonymous"
	loginUser.IsActive = false
	loginUser.Email = fmt.Sprintf("%s@%s", oldName,
		time.Now().Format("20060102150405"))
	loginUser.Update()

	// ログ登録
	var logRecord model.Log
	logRecord.UserId = loginUser.ID
	logRecord.Action = "withdrawal"
	logRecord.IpAddress = c.ClientIP()
	logRecord.CreateLog()

	//セッションからデータを破棄する
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{Path: "/", MaxAge: -1})
	session.Save()

	utils.DeleteCookie(c, "token")
	c.SetCookie("withdrawal-success", "true", 180, "", "", true, true)

	c.Redirect(302, "/finish-withdrawal")
}

func validatePassword(pass string) error {
	if len(pass) < 8 {
		return errors.New("パスワードは8文字以上で書いてください。")
	}
	if regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(pass) {
		return errors.New("パスワードは英数字記号を含めてください。")
	}
	if regexp.MustCompile(`^[0-9]+$`).MatchString(pass) {
		return errors.New("パスワードは英数字記号を含めてください。")
	}
	if regexp.MustCompile(`^[\-^$*.@]+$`).MatchString(pass) {
		return errors.New("パスワードは英数字記号を含めてください。")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9\-^$*.@]+$`).MatchString(pass) {
		return errors.New("パスワードは英数字記号で設定してください。")
	}

	return nil
}
