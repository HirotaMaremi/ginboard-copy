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
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// VOTE一覧画面
func ListVote(c *gin.Context) {
	param := strings.TrimSpace(c.Query("param"))
	tagParam := c.Query("tag")
	url := c.Request.URL.Path
	//requestUrl := c.Request.URL.RequestURI() // パラメータ付きURL
	linkPath := url + "&param=" + param + "&"

	thCount, err := model.CountVotesByParam(param, tagParam)
	if err != nil {
		log.Println(err.Error())
	}
	page := utils.ConvertPage(c, int(thCount))
	threads, err2 := model.GetVotesPagination(page, param, tagParam)
	if err2 != nil {
		log.Println(err2.Error())
	}

	// CloudFront path
	cloudFrontDomain := os.Getenv("AWS_CLOUDFRONT_DOMAIN") + "/" + os.Getenv("ENV")

	// ページネーションの頁番号表示のための配列
	pages := make([]int, page.TotalPages, page.TotalPages)
	for i := 0; i < page.TotalPages; i++ {
		pages[i] = i + 1
	}
	now := time.Now()
	template := "pc/vote_list.html"
	if utils.IsSp(c) {
		template = "sp/vote_list.html"
	}
	Render(c, 200, template, gin.H{
		"title":      "VOTE",
		"votes":      threads,
		"pagination": page,
		"pages":      pages,
		"linkPath":   linkPath,
		"param":      param,
		"now":        now,
		"imgDomain":  cloudFrontDomain,
	})
}

// VOTE 閲覧画面
func ShowOneVote(c *gin.Context) {
	num := c.Param("num")
	thread, err := model.GetOneVoteAllAssociationsByNum(num)
	if err != nil {
		log.Println(err.Error())
		c.Error(errors.New("スレッドが存在しません。")).SetType(gin.ErrorTypePublic)
		return
	}

	vote, _ := model.GetVoteByThread(thread.ID)
	voteOptions, _ := model.GetVoteOptionsByThread(thread.ID)

	// ログインユーザが既に投票したかどうか
	isVoted := true
	loginUserId, _ := c.Get("userID")
	_, err2 := model.GetUserVoteByUserThread(uint(loginUserId.(float64)), thread.ID)
	if errors.Is(err2, gorm.ErrRecordNotFound) {
		isVoted = false
	}

	midtermResultsMap, strCountedDates := createMidtermResultMap(vote.ID)

	viewModel, err := service.CreateThreadViewModel(thread, c)
	if err != nil {
		c.Error(err).SetType(gin.ErrorTypePublic)
		return
	}

	url := c.Request.URL.Path
	token := csrf.GetToken(c)
	// chart.js用データ作成
	var labels []string
	var data []uint
	for _, vo := range voteOptions {
		labels = append(labels, vo.Name)
		data = append(data, vo.Count)
	}

	hasDetail := false
	if thread.Detail.Valid == true && thread.Detail.String != "" {
		hasDetail = true
	}

	imageUrl := ""
	if thread.ImageFile != nil {
		imageUrl = utils.GetImageUrl(thread.ImageFile)
	}
	template := "pc/vote_show.html"
	if utils.IsSp(c) {
		template = "sp/vote_show.html"
	}
	Render(c, 200, template, gin.H{
		"title":             thread.Title,
		"thread":            thread,
		"comments":          viewModel.Comments,
		"token":             token,
		"pagination":        viewModel.Pagination,
		"pages":             viewModel.Pages,
		"likeCount":         viewModel.LikeCount,
		"url":               url,
		"userMap":           viewModel.UserMap,
		"goodCommentIds":    viewModel.GoodCommentIds,
		"likeMap":           viewModel.LikeMap,
		"vote":              vote,
		"voteOptions":       voteOptions,
		"isVoted":           isVoted,
		"midtermResultsMap": midtermResultsMap,
		"strCountedDates":   strCountedDates,
		"labels":            labels,
		"chartData":         data,
		"imageUrl":          imageUrl,
		"hasDetail":         hasDetail,
	})
}

// 作成・編集画面
func ShowCreateVote(c *gin.Context) {
	num := c.Param("threadNum")

	isEdit := false
	title := "VOTE作成"
	var thread model.Thread
	var vote model.Vote
	var voteOptions []model.VoteOption

	if num != "" {
		dbThread, err := model.GetOneVoteThreadByNum(num)
		if err != nil {
			log.Println(err.Error())
			c.Error(errors.New("該当のスレッドが存在しません。")).SetType(gin.ErrorTypePublic)
			return
		}
		isEdit = true
		title = "VOTE編集"
		thread = dbThread

		vote, _ = model.GetVoteByThread(thread.ID)
		voteOptions, _ = model.GetVoteOptionsByThread(dbThread.ID)
	}

	midtermResultsMap, strCountedDates := createMidtermResultMap(vote.ID)

	token := csrf.GetToken(c)
	Render(c, 200, "admin/create_vote.html", gin.H{
		"token":             token,
		"title":             title,
		"thread":            thread,
		"isEdit":            isEdit,
		"vote":              vote,
		"options":           voteOptions,
		"midtermResultsMap": midtermResultsMap,
		"strCountedDates":   strCountedDates,
		"now":               time.Now(),
	})
}

func CreateVote(c *gin.Context) {
	token := csrf.GetToken(c)
	num := c.Param("threadNum")
	loginUserId, _ := c.Get("userID")
	var thread model.Thread
	var vote model.Vote
	var voteOptions []model.VoteOption
	isEdit := false
	title := "VOTE作成"
	if num != "" {
		dbThread, err := model.GetOneVoteThreadByNum(num)
		if err != nil {
			c.Error(errors.New("該当のスレッドが存在しません。")).SetType(gin.ErrorTypePublic)
			return
		}
		isEdit = true
		title = "VOTE編集"
		thread = dbThread
		vote, _ = model.GetVoteByThread(thread.ID)
		voteOptions, _ = model.GetVoteOptionsByThread(dbThread.ID)
	} else {
		thread.Type = sql.NullString{"vote", true}
		thread.ThreadNumber = xid.New().String()
		thread.CreatedBy = uint(loginUserId.(float64))
	}

	thread.Title = c.PostForm("title")
	thread.Description = c.PostForm("description")
	if c.PostForm("detail") != "" {
		detail := c.PostForm("detail")
		thread.Detail.String = detail
	}
	thread.IsValid, _ = strconv.ParseBool(c.PostForm("is_valid"))
	thread.Status = 1

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
		Render(c, 400, "admin/create_vote.html", gin.H{
			"token":   token,
			"title":   title,
			"status":  "error",
			"msg":     validationErr.Error(),
			"thread":  thread,
			"isEdit":  isEdit,
			"vote":    vote,
			"options": voteOptions,
		})
		return
	}
	// optionの重複チェック
	postedOptions := c.PostFormArray("option_name")
	if !isValidPostedVoteOption(postedOptions) {
		Render(c, 400, "admin/create_vote.html", gin.H{
			"token":   token,
			"title":   title,
			"status":  "error",
			"msg":     "選択肢が重複しています。",
			"thread":  thread,
			"isEdit":  isEdit,
			"vote":    vote,
			"options": voteOptions,
		})
		return
	}

	voteType, _ := strconv.ParseUint(c.PostForm("vote-type"), 10, 64)
	vote.Type = uint(voteType)
	if c.PostForm("start_date") != "" {
		parsedStartAt, _ := time.Parse("2006-01-02T15:04", c.PostForm("start_date"))
		vote.StartAt = &parsedStartAt
	}
	if c.PostForm("end_date") != "" {
		parsedEndAt, _ := time.Parse("2006-01-02T15:04", c.PostForm("end_date"))
		vote.EndAt = &parsedEndAt
	}
	// voteOptionをDB保存するために先にthread, voteをDB保存する
	if isEdit {
		thread.UpdateThread()
		vote.ThreadId = thread.ID
		vote.UpdateVote()
	} else {
		thread.CreateThread()
		vote.ThreadId = thread.ID
		vote.CreateVote()
	}

	//VoteOptionの処理
	var deleteVoteOptions []model.VoteOption
	var createVoteOptions []model.VoteOption
	var voteOptionsName []string
	for _, option := range voteOptions {
		voteOptionsName = append(voteOptionsName, option.Name)
		if !utils.StrContains(postedOptions, option.Name) {
			deleteVoteOptions = append(deleteVoteOptions, option)
		}
	}
	for _, postedOp := range postedOptions {
		if !utils.StrContains(voteOptionsName, postedOp) {
			createVoteOptions = append(createVoteOptions, model.VoteOption{VoteId: vote.ID, Name: postedOp})
		}
	}

	hasError := false
	if len(createVoteOptions) != 0 {
		err := model.BulkCreateVoteOption(createVoteOptions)
		if err != nil {
			hasError = true
			log.Println(err.Error())
		}
	}
	if len(deleteVoteOptions) != 0 {
		err := model.BulkDeleteVoteOption(deleteVoteOptions)
		if err != nil {
			hasError = true
			log.Println(err.Error())
		}
	}
	if hasError {
		Render(c, 400, "admin/create_vote.html", gin.H{
			"token":   token,
			"title":   title,
			"status":  "error",
			"msg":     "VoteOption保存中にエラー",
			"thread":  thread,
			"isEdit":  isEdit,
			"vote":    vote,
			"options": voteOptions,
		})
		return
	}

	c.Redirect(302, "/admin")
}

// 投票
func VoteAction(c *gin.Context) {
	optionId := c.PostForm("option_id")
	threadId := c.PostForm("thread_id")
	fmt.Println(optionId)
	optionIdUint, _ := strconv.ParseUint(optionId, 10, 64)
	threadIdUint, _ := strconv.ParseUint(threadId, 10, 64)
	var userVote model.UserVote
	userVote.UserId = utils.GetLoginUserId(c)
	userVote.VoteOptionId = uint(optionIdUint)
	userVote.ThreadId = uint(threadIdUint)
	userVote.CreateUserVote()

	c.JSON(200, gin.H{
		"success": true,
	})
}

// 中間集計
func CountMidtermResult(c *gin.Context) {
	threadId := c.PostForm("thread_id")
	threadIdUint, _ := strconv.ParseUint(threadId, 10, 64)
	userVoteCounts, _ := model.AggregateUserVoteByThreadId(uint(threadIdUint))
	for _, count := range userVoteCounts {
		midtermResult := new(model.VoteMidtermResult)
		midtermResult.Count = count.Count
		midtermResult.VoteOptionId = count.OptionID
		midtermResult.CountedAt = time.Now()
		midtermResult.CreateVoteMidtermResult()
	}

	// 投票者へ通知を送信
	thread, _ := model.GetOneThreadById(uint(threadIdUint))
	err := model.CreateForVotersByThreadId(uint(threadIdUint), "中間集計結果", "あなたが投票したVOTEの中間集計が行われました。<br/>結果は<a href='/member/vote/show/"+thread.ThreadNumber+"'>こちら</a>。")
	if err != nil {
		log.Println(err.Error())
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	c.JSON(200, gin.H{
		"success": true,
	})
}

// 最終集計
func CountFinalResult(c *gin.Context) {
	threadId := c.PostForm("thread_id")
	threadIdUint, _ := strconv.ParseUint(threadId, 10, 64)
	vote, _ := model.GetVoteByThread(uint(threadIdUint))
	userVoteCounts, _ := model.AggregateUserVoteByThreadId(uint(threadIdUint))
	for _, count := range userVoteCounts {
		voteOption, err := model.GetVoteOptionById(count.OptionID)
		if err != nil {
			c.Error(err).SetType(gin.ErrorTypePublic)
			return
		}
		voteOption.Count = count.Count
		voteOption.UpdateVoteOption()
	}

	now := time.Now()
	vote.CountedAt = &now
	vote.UpdateVote()

	// 投票者へ通知を送信
	thread, _ := model.GetOneThreadById(uint(threadIdUint))
	err := model.CreateForVotersByThreadId(uint(threadIdUint), "最終集計結果", "あなたが投票したVOTEの最終集計が行われました。<br/>結果は<a href='/member/vote/show/"+thread.ThreadNumber+"'>こちら</a>。")
	if err != nil {
		log.Println(err.Error())
		c.Error(errors.New("エラーが発生しました。")).SetType(gin.ErrorTypePublic)
		return
	}

	c.JSON(200, gin.H{
		"success": true,
	})
}

func isValidPostedVoteOption(options []string) bool {
	length := len(options)
	unique := utils.RemoveStrArrayDuplicate(options)

	return length == len(unique)
}

func createMidtermResultMap(voteId uint) (map[string]service.ByRank, []string) {
	// 中間集計結果(集計日時をキーに各候補者の得票数を持つ)
	midtermResultsMap := make(map[string]service.ByRank)
	// 日付ソート用配列
	var strCountedDates []string
	midtermResults, _ := model.GetVoteMidtermResultByThread(voteId)
	for _, r := range midtermResults {
		strCountedAt := r.CountedAt.Format("2006/01/02 15:04:05")
		if !utils.StrContains(strCountedDates, strCountedAt) {
			strCountedDates = append(strCountedDates, strCountedAt)
		}
		optionId := r.VoteOptionId
		option, _ := model.GetVoteOptionById(optionId)
		candidate := service.Candidate{Name: option.Name, Count: int(r.Count)}
		if len(midtermResultsMap[strCountedAt]) == 0 {
			byRank := service.ByRank{candidate}
			midtermResultsMap[strCountedAt] = byRank
		} else {
			midtermResultsMap[strCountedAt] = append(midtermResultsMap[strCountedAt], candidate)
		}
	}
	// 得票数でソート
	for _, m := range midtermResultsMap {
		sort.Sort(m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(strCountedDates)))

	return midtermResultsMap, strCountedDates
}
