package controller

import (
	"errors"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
	"log"
	"os"
	"regexp"
)

func About(c *gin.Context) {
	Render(c, 200, "about.html", gin.H{
		"title": "About",
	})
}

func Help(c *gin.Context) {
	Render(c, 200, "help.html", gin.H{
		"title": "ヘルプ",
	})
}

func History(c *gin.Context) {
	Render(c, 200, "history.html", gin.H{
		"title": "開発経緯",
	})
}

func Inquiry(c *gin.Context) {
	token := csrf.GetToken(c)
	Render(c, 200, "inquiry.html", gin.H{
		"title": "お問い合わせ",
		"token": token,
	})
}

func SendInquiry(c *gin.Context) {
	token := csrf.GetToken(c)
	name := c.PostForm("name")
	email := c.PostForm("email")
	inquiry := c.PostForm("inquiry")

	if !regexp.MustCompile(`^[ぁ-んァ-ヶー一-龯々]+$`).MatchString(name) {
		Render(c, 400, "inquiry.html", gin.H{
			"title":   "お問い合わせ",
			"msg":     "氏名は日本語で入力してください。",
			"success": false,
			"token":   token,
		})
		return
	}

	if len([]rune(inquiry)) > 400 {
		Render(c, 400, "inquiry.html", gin.H{
			"title":   "お問い合わせ",
			"msg":     "お問い合わせは400文字以内で記載ください。",
			"success": false,
			"token":   token,
		})
		return
	}

	err := utils.SendMailInquiry(name, inquiry, email, c.ClientIP(), os.Getenv("ENV"))
	if err != nil {
		log.Println(err.Error())
		Render(c, 400, "inquiry.html", gin.H{
			"title":   "お問い合わせ",
			"msg":     "送信できませんでした。しばらく時間をおいてから再度試してください。",
			"success": false,
			"token":   token,
		})
		return
	}

	// ログ登録
	var log model.Log
	log.UserId = 0
	log.Action = "inquiry"
	log.IpAddress = c.ClientIP()
	log.CreateLog()
	Render(c, 200, "inquiry.html", gin.H{
		"title":   "お問い合わせ",
		"msg":     "お問い合わせを受け付けました。",
		"success": true,
		"token":   token,
	})
}

func RequestAccount(c *gin.Context) {
	token := csrf.GetToken(c)
	Render(c, 200, "request_account.html", gin.H{
		"title": "メンバー登録リクエスト",
		"token": token,
	})
}

func SendRequestAccount(c *gin.Context) {
	token := csrf.GetToken(c)
	email := c.PostForm("email")
	snsType := c.PostForm("sns")
	snsAccount := c.PostForm("name")
	note := c.PostForm("note")

	if len([]rune(note)) > 100 {
		Render(c, 100, "request_account.html", gin.H{
			"title":   "メンバー登録リクエスト",
			"msg":     "備考は100文字以内で記載ください。",
			"success": false,
			"token":   token,
		})
		return
	}

	var sns string
	switch snsType {
	case "1":
		sns = "twitter"
	case "2":
		sns = "Instagram"
	case "3":
		sns = "Facebook"
	case "4":
		sns = "その他"
	}

	err := utils.SendMailRequestAccount(snsAccount, sns, note, email)
	if err != nil {
		log.Println(err.Error())
		Render(c, 400, "request_account.html", gin.H{
			"title":   "メンバー登録リクエスト",
			"msg":     "送信できませんでした。しばらく時間をおいてから再度試してください。",
			"success": false,
			"token":   token,
		})
		return
	}

	Render(c, 200, "request_account.html", gin.H{
		"title":   "メンバー登録リクエスト",
		"msg":     "メンバー登録リクエストを受け付けました。",
		"success": true,
		"token":   token,
	})
}

func Privacy(c *gin.Context) {
	token := csrf.GetToken(c)
	Render(c, 200, "privacy.html", gin.H{
		"title": "プライバシーポリシー",
		"token": token,
	})
}

func FinishWithdrawal(c *gin.Context) {
	success, err := c.Cookie("withdrawal-success")
	if err != nil {
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	Render(c, 200, "withdrawal-finish.html", gin.H{
		"title":   "退会完了",
		"success": success,
	})
}

func GoogleAdsTxt(c *gin.Context) {
	c.Data(200, "text/plain; charset=utf-8", []byte("google.com, pub-8138427809816157, DIRECT, f08c47fec0942fa0"))
}

func Gallery(c *gin.Context) {
	token := csrf.GetToken(c)

	template := "pc/gallery.html"
	if utils.IsSp(c) {
		template = "sp/gallery.html"
	}

	Render(c, 200, template, gin.H{
		"title": "ギャラリー",
		"token": token,
	})
}
