package controller

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/casbin/casbin/v2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
	"gorm.io/gorm"
	"log"
	"net"
	"regexp"
	"strconv"
	"time"
)

// メンバー登録画面表示
func showRegister(c *gin.Context) {
	//https://pkg.go.dev/github.com/withgame/gin-csrf
	token := csrf.GetToken(c)
	Render(c, 200, "register.html", gin.H{"token": token, "title": "メンバー登録"})
}

// ログイン画面表示
func showSignIn(c *gin.Context) {
	token := csrf.GetToken(c)
	Render(c, 200, "sign_in.html", gin.H{"token": token, "title": "ログイン"})
}

// メンバー登録処理
func AddUser(enforcer *casbin.Enforcer) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := csrf.GetToken(c)
		var user model.User
		// https://gin-gonic.com/ja/docs/examples/binding-and-validation/
		if err := c.ShouldBind(&user); err != nil {
			Render(c, 400, "register.html", gin.H{
				"token":  token,
				"status": "warning",
				"msg":    err.Error(),
				"user":   user,
			})
			return
		}

		// 紹介コードの検証
		code := c.PostForm("code")
		inviteCode, err := model.FindValidInviteCode(code)
		if err != nil {
			// todo: err.Error()の出力
			Render(c, 400, "register.html", gin.H{
				"token":  token,
				"status": "warning",
				"msg":    "紹介コードが不正です。",
				"user":   user,
				"code":   code,
			})
			return
		}
		now := time.Now()
		inviteUserId := inviteCode.UserId
		inviteCode.UsedAt = &now

		// 入力された２件のパスワードの一致確認
		pass := c.PostForm("password")
		pass2 := c.PostForm("password2")
		if pass != pass2 {
			Render(c, 400, "register.html", gin.H{
				"token":  token,
				"status": "warning",
				"msg":    "パスワードが一致していません。",
				"user":   user,
				"code":   code,
			})
			return
		}
		validationErr := validate(user)
		if validationErr != nil {
			Render(c, 400, "register.html", gin.H{
				"token":  token,
				"status": "warning",
				"msg":    validationErr.Error(),
				"user":   user,
			})
			//c.Abort()
			return
		}

		// パスワードはハッシュ化して登録
		utils.HashPassword(&user.Password)

		lastUser := model.GetLastMemberNumberOne()
		lastMemberNum, _ := strconv.Atoi(lastUser.MemberNumber)
		user.MemberNumber = strconv.Itoa(lastMemberNum + 1)
		user.IsActive = false
		user.InvitedBy = &inviteUserId

		// 認証トークンの保存
		confirmationToken := createConfirmationToken(lastMemberNum)
		user.ConfirmationToken = confirmationToken
		// 有効期限は24時間
		expireDate := time.Now().AddDate(0, 0, 1)
		user.TokenExpireAt = &expireDate

		// ユーザ登録
		user.Create()
		// 権限登録
		enforcer.AddGroupingPolicy(fmt.Sprint(user.ID), "member")
		// 紹介コード更新
		inviteCode.UpdateInviteCode()
		// ログ登録
		var log model.Log
		log.UserId = user.ID
		log.Action = "add user"
		log.IpAddress = c.ClientIP()
		log.CreateLog()

		url := utils.GetRootUrl(c) + "/confirm?token=" + user.ConfirmationToken + "&id=" + fmt.Sprint(user.ID)
		// 確認メール送信
		utils.SendMail(url, user.Email)

		user.Password = ""
		//c.JSON(http.StatusOK, user)
		Render(c, 200, "register.html", gin.H{
			"success": true,
			"status":  "success",
			"msg":     "仮登録が完了しました。確認メールを送信いたしましたので記載のURLから本登録を完了させてください。",
		})
	}
}

func SignInUser(c *gin.Context) {
	var user model.User

	// 不正IPアドレスでないか。
	if existInBlacklistIpAddress(c.ClientIP()) {
		log.Println("ブラックリストIPからのログイン試行" + c.ClientIP())
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	if err := c.ShouldBind(&user); err != nil {
		log.Println(err.Error())
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		//c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session := sessions.Default(c)
	token := csrf.GetToken(c)

	loginCnt := 0
	cnt, cErr := c.Cookie("cnt")
	if cErr == nil {
		loginCnt, _ = strconv.Atoi(cnt)
	}
	if loginCnt > 5 {
		c.Error(errors.New("ログイン試行回数が上限に達しました。しばらくたってから再度お試しください。")).SetType(gin.ErrorTypePublic)

		return
	}

	var dbUser model.User
	var err2 error
	if user.Name != "" {
		dbUser, err2 = model.GetByName(user.Name)
	} else if user.Email != "" {
		dbUser, err2 = model.GetByEmail(user.Email)
	}

	if err2 != nil {
		// ログ登録
		var dbLog model.Log
		dbLog.UserId = dbUser.ID // = 0
		dbLog.Action = "wrong sign in"
		dbLog.IpAddress = c.ClientIP()
		dbLog.CreateLog()

		c.SetCookie("cnt", fmt.Sprint(loginCnt+1), 3600*3*1, "", "", true, true)
		session.AddFlash("ログイン情報が不正です")
		Render(c, 200, "sign_in.html", gin.H{
			"token":    token,
			"messages": session.Flashes(),
		})
		return
	}

	if !dbUser.IsActive {
		var dbLog model.Log
		dbLog.UserId = dbUser.ID
		dbLog.Action = "wrong sign in, not activated"
		dbLog.IpAddress = c.ClientIP()
		dbLog.CreateLog()

		c.SetCookie("cnt", fmt.Sprint(loginCnt+1), 3600*3*1, "", "", true, true)
		session.AddFlash("ログイン情報が不正です")
		Render(c, 200, "sign_in.html", gin.H{
			"token":    token,
			"messages": session.Flashes(),
		})
		return
	}

	if isTrue := utils.ComparePassword(dbUser.Password, user.Password); isTrue {
		token := utils.GenerateToken(dbUser.ID)
		// Cookieにトークンをセット
		utils.SetCookie(c, "token", token)
		tz, _ := time.LoadLocation("Asia/Tokyo")
		now := time.Now().In(tz)
		dbUser.LastLoggedInAt = &now
		dbUser.Update()

		// ログ登録
		var dbLog model.Log
		dbLog.UserId = dbUser.ID
		dbLog.Action = "signed in"
		dbLog.IpAddress = c.ClientIP()
		dbLog.CreateLog()

		utils.DeleteCookie(c, "cnt")

		c.Redirect(302, "/member")
		return
	}

	c.SetCookie("cnt", fmt.Sprint(loginCnt+1), 3600*3*1, "", "", true, true)
	session.AddFlash("ログイン情報が不正です")
	Render(c, 200, "sign_in.html", gin.H{
		"token":    token,
		"messages": session.Flashes(),
	})

	return

}

func Logout(c *gin.Context) {
	//セッションからデータを破棄する
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{Path: "/", MaxAge: -1})
	session.Save()

	utils.DeleteCookie(c, "token")

	c.Redirect(302, "/")
}

// ユーザ登録確認メールのエンドポイント
func VerifyConfirmationToken(c *gin.Context) {
	//token := c.Param("token")
	token := c.Query("token")
	userId := c.Query("id")
	id, _ := strconv.Atoi(userId)
	user, err := model.GetConfirmationUser(id, token)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		Render(c, 200, "register_confirmation.html", gin.H{
			"message": "URLの有効期限が切れています。",
			"success": false,
		})
		return
	}
	if err != nil {
		log.Println(err.Error())
		Render(c, 400, "register_confirmation.html", gin.H{
			"message": "エラーが発生しました。",
			"success": false,
		})
		return
	}

	user.IsActive = true
	user.Update()
	Render(c, 200, "register_confirmation.html", gin.H{
		"message": "本登録が完了しました！",
		"success": true,
	})
}

func validate(user model.User) error {
	if len(user.Password) < 8 {
		return errors.New("パスワードは8文字以上で書いてください。")
	}
	if regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(user.Password) {
		return errors.New("パスワードは英数字記号を含めてください。")
	}
	if regexp.MustCompile(`^[0-9]+$`).MatchString(user.Password) {
		return errors.New("パスワードは英数字記号を含めてください。")
	}
	if regexp.MustCompile(`^[\-^$*.@]+$`).MatchString(user.Password) {
		return errors.New("パスワードは英数字記号を含めてください。")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9\-^$*.@]+$`).MatchString(user.Password) {
		return errors.New("パスワードは英数字記号で設定してください。")
	}
	if len([]rune(user.Name)) >= 200 {
		return errors.New("ユーザー名は200文字未満で書いてください。")
	}
	if len([]rune(user.Name)) < 5 {
		return errors.New("ユーザー名は5文字以上にしてください。")
	}
	if user.Name == "anonymous" {
		return errors.New("このユーザ名は使用できません。")
	}
	_, err := model.GetByEmail(user.Email)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("入力いただいた内容での登録はできません。")
	}
	// nameはPGでのユニーク制約のみ。DB上はユニーク制約無し
	_, err2 := model.GetByName(user.Name)
	if !errors.Is(err2, gorm.ErrRecordNotFound) {
		return errors.New("入力いただいた内容での登録はできません。")
	}
	return nil
}

func createConfirmationToken(lastMemberNum int) string {
	now := time.Now()
	nanos := now.UnixNano()
	p := []byte(fmt.Sprint(nanos) + fmt.Sprint(lastMemberNum))
	sha := sha256.Sum256(p)
	return fmt.Sprintf("%x", sha)
}

func existInBlacklistIpAddress(ip string) bool {
	// todo: いずれブラックリストはDBで管理する
	blacklist := []string{"5.188.62.26", "5.188.62.140"}
	blacklistCidr := []string{"5.188.62.0/23"} // ロシア不正ログイン

	for _, val := range blacklist {
		if val == ip {
			return true
		}
	}
	for _, cidr := range blacklistCidr {
		if existInBlacklistCidr(ip, cidr) {
			return true
		}
	}
	return false
}

func existInBlacklistCidr(ip string, cidr string) bool {
	// todo: いずれブラックリストはDBで管理する
	_, ipnet, _ := net.ParseCIDR(cidr)
	ip1 := net.ParseIP(ip)

	return ipnet.Contains(ip1)
}
