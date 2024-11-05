package controller

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/service"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/gin-gonic/gin"
	"github.com/olahol/go-imageupload"
	"gorm.io/gorm"
	"log"
	"os"
	"strconv"
	"time"
)

func ApiPostComment(c *gin.Context) {
	// コメント投稿時、ユーザとスレッドのLastCommentedAtも更新する
	var comment model.Comment
	if err := c.ShouldBind(&comment); err != nil {
		c.JSON(400, gin.H{
			"success": false,
		})
		return
	}
	// 返信かどうかのフラグ
	isReply := comment.ParentId != nil

	lastThread := model.GetLastComment(comment.ThreadId)
	loginUserId, _ := c.Get("userID")
	comment.CommentNumber = lastThread.CommentNumber + 1
	comment.CreatedBy = uint(loginUserId.(float64))
	comment.IsValid = true
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

	// 返信の場合、コメ主に通知を送る
	if isReply {
		//referer := c.Request.Header.Get("Referer")
		parentId := comment.ParentId
		parentComment := model.GetOneComment(*parentId)
		commentUrl := ""
		switch thread.Type.String {
		case "normal":
			commentUrl = "/member/show/" + fmt.Sprint(thread.ThreadNumber) + "?comment=" + fmt.Sprint(comment.ID)
		case "vote":
			commentUrl = "/member/vote/show/" + fmt.Sprint(thread.ThreadNumber) + "?comment=" + fmt.Sprint(comment.ID)
		case "private":
			commentUrl = "/private/show/" + fmt.Sprint(thread.ThreadNumber) + "?comment=" + fmt.Sprint(comment.ID)
		case "giin_private":
			commentUrl = "/private/giinShow/" + fmt.Sprint(thread.ThreadNumber) + "?comment=" + fmt.Sprint(comment.ID)
		}
		var notice model.Notice
		notice.UserId = parentComment.CreatedBy
		notice.Title = "あなたのコメントに返信がありました。"
		notice.Body = "あなたへのコメントに対する返信は<br/><a href='" + commentUrl + "'>こちら</a>。"
		notice.IsValid = true
		notice.CreateNotice()
	}

	// ログ登録
	var log model.Log
	log.UserId = loginUser.ID
	if isReply {
		log.Action = "reply comment"
	} else {
		log.Action = "post comment"
	}
	log.IpAddress = c.ClientIP()
	log.CreateLog()

	c.JSON(200, gin.H{
		"success": true,
	})
}

func ApiPostGood(c *gin.Context) {
	// いいね機能
	var like model.Like
	if err := c.ShouldBind(&like); err != nil {
		c.JSON(400, gin.H{
			"success1": false,
		})
		return
	}

	commentId := like.CommentId

	loginUserId, _ := c.Get("userID")
	existingLike, err := model.FindLikeByCommentAndUser(commentId, uint(loginUserId.(float64)))
	isInsert := false
	needDelete := true

	if errors.Is(err, gorm.ErrRecordNotFound) {
		//既存レコードが無い
		isInsert = true
		needDelete = false
	} else if err != nil {
		c.JSON(400, gin.H{
			"success": false,
		})
		return
	} else if like.Good == existingLike.Good {
		// レコードの削除
		isInsert = false
		needDelete = true

	} else {
		// good <-> badの変更
		isInsert = true
		needDelete = true

	}

	if needDelete {
		//いいねレコード削除
		existingLike.DeleteLike()

		if !isInsert {
			// いいね内訳取得
			likeCount, err := model.CountByCommentId(commentId)
			if err != nil {
				c.JSON(400, gin.H{
					"success": false,
				})
				return
			}

			// ログ登録
			var log model.Log
			log.UserId = uint(loginUserId.(float64))
			log.Action = "delete like"
			log.IpAddress = c.ClientIP()
			log.CreateLog()

			c.JSON(200, gin.H{
				"success":       true,
				"goodCount":     likeCount.GoodCount,
				"badCount":      likeCount.BadCount,
				"isChangedLike": isInsert && needDelete,
			})
			return
		}
	}

	like.UserId = uint(loginUserId.(float64))
	like.CommentId = commentId

	like.CreateLike()
	// いいね内訳取得
	likeCount, err := model.CountByCommentId(commentId)
	if err != nil {
		c.JSON(400, gin.H{
			"success1": false,
		})
		return
	}

	// ログ登録
	var log model.Log
	log.UserId = uint(loginUserId.(float64))
	log.Action = "post like"
	log.IpAddress = c.ClientIP()
	log.CreateLog()

	c.JSON(200, gin.H{
		"success":       true,
		"goodCount":     likeCount.GoodCount,
		"badCount":      likeCount.BadCount,
		"isChangedLike": isInsert && needDelete,
	})
}

func ApiImageUpload(c *gin.Context) {
	//imageupload.LimitFileSize(1024*1024, c.Writer, c.Request)
	img, err := imageupload.Process(c.Request, "upload")
	postedToken := c.PostForm("ckCsrfToken")
	cookieToken, err2 := c.Cookie("ckCsrfToken")

	if err != nil || err2 != nil || postedToken != cookieToken {
		// 画像ファイルが設定されていない
		c.JSON(400, gin.H{
			"uploaded": false,
		})
		return
	}
	//thumb, err := imageupload.ThumbnailPNG(img, 500, 500)
	//if err != nil {
	//	c.JSON(400, gin.H{
	//		"uploaded": false,
	//	})
	//	return
	//}

	dir := "file/thread/"
	errDir := utils.CreateFolder(dir)
	if errDir != nil {
		c.Error(errDir).SetType(gin.ErrorTypePublic)
		c.JSON(400, gin.H{
			"uploaded": false,
		})
		return
	}
	h := sha1.Sum(img.Data)
	filename := fmt.Sprintf("%s_%x.png",
		time.Now().Format("20060102150405"), h[:4])
	err3 := img.Save(dir + "/" + filename)
	if err3 != nil {
		c.Error(err3).SetType(gin.ErrorTypePublic)
		c.JSON(400, gin.H{
			"uploaded": false,
		})
		return
	}
	// S3Upload 5回までリトライ
	for i := 0; i < 5; i++ {
		err := service.UploadS3(dir+"/"+filename, os.Getenv("ENV")+"/thread/"+filename)
		if err == nil {
			os.Remove(dir + "/" + filename)
			break
		}
	}

	cloudFrontDomain := os.Getenv("AWS_CLOUDFRONT_DOMAIN") + "/" + os.Getenv("ENV")
	c.JSON(200, gin.H{
		"uploaded": true,
		"url":      cloudFrontDomain + "/thread/" + filename,
	})
}

func ApiGiinPostComment(c *gin.Context) {
	// コメント投稿時、ユーザとスレッドのLastCommentedAtも更新する
	var giinItem model.GiinItem
	if err := c.ShouldBind(&giinItem); err != nil {
		c.JSON(400, gin.H{
			"success": false,
		})
		return
	}

	loginUserId, _ := c.Get("userID")
	giinItem.CreatedBy = uint(loginUserId.(float64))
	giinItem.CreateGiinItem()

	// ユーザのLastCommentedAt更新
	loginUser := model.GetOne(int(loginUserId.(float64)))
	now := time.Now()
	loginUser.LastCommentedAt = &now
	loginUser.Update()

	// ログ登録
	var log model.Log
	log.UserId = loginUser.ID
	log.Action = "post giin item"
	log.IpAddress = c.ClientIP()
	log.CreateLog()

	c.JSON(200, gin.H{
		"success": true,
	})
}

func ApiGiinDeleteComment(c *gin.Context) {
	itemId := c.PostForm("item_id")
	createdBy, _ := strconv.Atoi(c.PostForm("created_by"))
	loginUserId, _ := c.Get("userID")

	if itemId == "" || createdBy != int(loginUserId.(float64)) {
		log.Println("--ApiGiinDeleteComment--")
		log.Println(itemId)
		log.Println(createdBy)
		c.JSON(400, gin.H{
			"success": false,
		})
		return
	}

	intItemId, _ := strconv.Atoi(itemId)
	giinItem, err := model.GetOneGiinItem(intItemId)
	if err != nil {
		log.Println(err.Error())
		c.JSON(400, gin.H{
			"success": false,
		})
		return
	}
	giinItem.DeleteGiinItem()

	// ログ登録
	var log model.Log
	log.UserId = uint(loginUserId.(float64))
	log.Action = "delete giin item"
	log.IpAddress = c.ClientIP()
	log.CreateLog()

	c.JSON(200, gin.H{
		"success": true,
	})
}
