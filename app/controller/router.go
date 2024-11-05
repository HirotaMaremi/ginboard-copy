package controller

import (
	"fmt"
	"github.com/HirotaMaremi/ginboard/middleware"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
	"gorm.io/gorm"
	"html/template"
	"log"
	"math"
	"os"
	"strings"
	"time"
)

// https://zenn.dev/ajapa/articles/65b9934db18396
func GetRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	store := cookie.NewStore([]byte("931cce107945e00cb2c102f13d36f63c"))
	store.Options(sessions.Options{MaxAge: 60 * 60 * 24})
	r.Use(sessions.Sessions("session", store))

	env := os.Getenv("ENV")

	// not found exception
	r.NoRoute(func(c *gin.Context) {
		c.HTML(404, "error.html", gin.H{"env": env})
		c.Abort()
	})

	// Initialize  casbin adapter
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize casbin adapter: %v", err))
	}

	// Load model configuration file and policy store adapter
	enforcer, err := casbin.NewEnforcer("config/rbac_model.conf", adapter)
	if err != nil {
		panic(fmt.Sprintf("failed to create casbin enforcer: %v", err))
	}

	//add policy
	if hasPolicy := enforcer.HasPolicy("anonymous", "board", "read"); !hasPolicy {
		enforcer.AddPolicy("anonymous", "board", "read")
	}
	if hasPolicy := enforcer.HasPolicy("member", "board", "write"); !hasPolicy {
		enforcer.AddPolicy("member", "board", "write")
	}
	if hasPolicy := enforcer.HasPolicy("admin", "board", "write"); !hasPolicy {
		enforcer.AddPolicy("admin", "board", "write")
	}

	r.Static("/assets", "./assets")

	r.SetFuncMap(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"raw": func(text string) template.HTML {
			return template.HTML(text)
		},
		"date":                  formatAsDate,
		"percentage":            division,
		"nl2br":                 nl2br,
		"hasField":              hasField,
		"formatAsDatetimeLocal": formatAsDatetimeLocal,
		"mod":                   mod,
	})
	r.LoadHTMLGlob("view/**/*.html")
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.HeaderHandler())

	userProtectedRoutes := r.Group("/member", middleware.AuthorizeJWT())
	{
		userProtectedRoutes.Use(csrf.Middleware(csrfOptions()))
		// スレ一覧
		userProtectedRoutes.GET("/", ShowTopThread)
		userProtectedRoutes.GET("/thr", ListThread)
		//userProtectedRoutes.GET("/thr/:param", ShowSearchAllThread)
		// 個別スレ画面
		userProtectedRoutes.GET("/show/:num", ShowOneThread)
		//userProtectedRoutes.POST("/show", middleware.Authorize("board", "write", enforcer), ApiPostComment)
		userProtectedRoutes.POST("/show/postComment", middleware.Authorize("board", "write", enforcer), ApiPostComment)
		userProtectedRoutes.POST("/show/postImageComment", middleware.Authorize("board", "write", enforcer), PostImageComment)
		userProtectedRoutes.POST("/show/postGood", middleware.Authorize("board", "write", enforcer), ApiPostGood)
		// スレッド編集
		userProtectedRoutes.GET("/create", middleware.Authorize("board", "write", enforcer), ShowCreateThread)
		userProtectedRoutes.POST("/create", CreateThread(enforcer))
		userProtectedRoutes.GET("/edit/:threadNum", ShowCreateThread)
		userProtectedRoutes.POST("/edit/:threadNum", CreateThread(enforcer))
		// VOTE
		userProtectedRoutes.GET("/vote", ListVote)
		userProtectedRoutes.GET("/vote/show/:num", ShowOneVote)
		userProtectedRoutes.POST("/vote/action", VoteAction)

		// post先のURLがGETと異なるとcsrfトークンが異なると言われてしまう
		//userProtectedRoutes.POST("/api/postComment", middleware.Authorize("board", "write", enforcer), ApiPostComment)
		//r.POST("/api/image-upload", ApiImageUpload)

	}
	apiRoutes := r.Group("/api")
	apiRoutes.POST("/image-upload", ApiImageUpload)

	accountRoutes := r.Group("/account", middleware.AuthorizeJWT())
	{
		accountRoutes.Use(csrf.Middleware(csrfOptions()))
		accountRoutes.GET("/", AccountTop)
		accountRoutes.POST("/changePassword", ChangePassword)
		accountRoutes.POST("/changeEmail", ChangeEmail)
		accountRoutes.GET("/inviteCode", ShowCode)
		accountRoutes.POST("/inviteCode/create", CreateCode)
		accountRoutes.GET("/threads", ListAccountThreads)
		accountRoutes.GET("/notices", ListNotices)
		accountRoutes.POST("/notices/viewed", ViewedNotice)
		accountRoutes.GET("/withdrawal", Withdrawal)
		accountRoutes.POST("/withdrawal", ExecuteWithdrawal)
		//todo: メアド変更
	}

	privateRoutes := r.Group("/private", middleware.AuthorizeJWT())
	{
		privateRoutes.Use(csrf.Middleware(csrfOptions()))
		// スレ一覧
		privateRoutes.GET("/", ShowPrivateThread)
		// 個別スレ画面
		privateRoutes.GET("/show/:num", ShowOnePrivateThread)
		privateRoutes.POST("/show/postComment", middleware.Authorize("board", "write", enforcer), ApiPostComment)
		privateRoutes.POST("/show/postGood", middleware.Authorize("board", "write", enforcer), ApiPostGood)
		privateRoutes.POST("/show/postImageComment", middleware.Authorize("board", "write", enforcer), PostImageComment)
		// 議員スレ一覧
		//議員スレは多数立たない想定なのでprivate一覧の下部に一覧追加する
		//privateRoutes.GET("/giin", ShowGiinThread)
		privateRoutes.GET("/giinShow/:num", ShowOneGiinThread)
		privateRoutes.POST("/giinPostItem", ApiGiinPostComment)
		privateRoutes.POST("/giinDeleteItem", ApiGiinDeleteComment)
		privateRoutes.POST("/giinShow/postComment", middleware.Authorize("board", "write", enforcer), ApiPostComment)
		privateRoutes.POST("/giinShow/postGood", middleware.Authorize("board", "write", enforcer), ApiPostGood)
		privateRoutes.POST("/giinShow/postImageComment", middleware.Authorize("board", "write", enforcer), PostImageComment)
	}

	adminRoutes := r.Group("/admin", middleware.AuthorizeJWT())
	{
		adminRoutes.Use(csrf.Middleware(csrfOptions()))
		adminRoutes.Use(middleware.AdminAuthorize(enforcer))
		adminRoutes.GET("/", AdminListThread)
		adminRoutes.GET("/thr", AdminListThread)
		adminRoutes.GET("/thr/edit/:threadNum", AdminShowCreateThread)
		adminRoutes.POST("/thr/edit/:threadNum", AdminCreateThread)
		adminRoutes.GET("/thr/create", AdminShowCreateThread)
		adminRoutes.POST("/thr/create", AdminCreateThread)
		adminRoutes.GET("/notice", AdminListNotice)
		adminRoutes.GET("/notice/edit/:id", AdminShowCreateNotice)
		adminRoutes.POST("/notice/edit/:id", AdminCreateNotice)
		adminRoutes.GET("/notice/create", AdminShowCreateNotice)
		adminRoutes.POST("/notice/create", AdminCreateNotice)
		adminRoutes.GET("/vote/create", ShowCreateVote)
		adminRoutes.POST("/vote/create", CreateVote)
		adminRoutes.GET("/vote/edit/:threadNum", ShowCreateVote)
		adminRoutes.POST("/vote/edit/:threadNum", CreateVote)
		adminRoutes.POST("/vote/midtermCount", CountMidtermResult)
		adminRoutes.POST("/vote/finalCount", CountFinalResult)

	}

	r.Use(csrf.Middleware(csrfOptions()))
	r.GET("/", ShowTopThread)
	r.GET("/thr", ListThread)
	r.GET("/show/:num", ShowOneThread)
	r.GET("/logout", Logout)
	r.GET("/register", showRegister)
	r.POST("/register", AddUser(enforcer))
	r.GET("/signin", showSignIn)
	r.POST("/signin", SignInUser)
	r.GET("/confirm", VerifyConfirmationToken)
	r.GET("/about", About)
	r.GET("/help", Help)
	r.GET("/history", History)
	r.GET("/inquiry", Inquiry)
	r.POST("/inquiry", SendInquiry)
	r.GET("/request", RequestAccount)
	r.POST("/request", SendRequestAccount)
	r.GET("/privacy", Privacy)
	r.GET("/finish-withdrawal", FinishWithdrawal)
	r.GET("/ads.txt", GoogleAdsTxt)
	r.GET("/gallery", Gallery)

	return r
}

func csrfOptions() csrf.Options {
	env := os.Getenv("ENV")
	csrfSecret := os.Getenv("CSRF_SECRET")
	return csrf.Options{
		Secret: csrfSecret,
		ErrorFunc: func(c *gin.Context) {
			log.Println("CSRF token mismatch!")
			c.HTML(500, "error.html", gin.H{"env": env})
			//c.String(400, "Security error!")
			c.Abort()
		},
	}
}

// Template関数
func formatAsDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	year, month, day := t.Date()
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second()
	return fmt.Sprintf("%d/%02d/%02d %02d:%02d:%02d", year, month, day, hour, minute, second)
}

func division(a, b uint) int {
	total := float64(a) + float64(b)
	if total == 0 {
		return 0
	}

	return int(math.Ceil(float64(a) / total * 100))
}

func mod(a, b int) int {
	return a % b
}

// Nl2br テンプレートで改行を有効にする
func nl2br(text string) template.HTML {
	return template.HTML(strings.Replace(template.HTMLEscapeString(text), "\n", "<br />", -1))
}

func hasField(a uint, array []uint) bool {
	for _, v := range array {
		if a == v {
			return true
		}
	}
	return false
}

func formatAsDatetimeLocal(t *time.Time) string {
	if t == nil {
		return ""
	}
	year, month, day := t.Date()
	hour := t.Hour()
	minute := t.Minute()
	second := t.Second()
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d", year, month, day, hour, minute, second)
}
