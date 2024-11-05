package service

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/HirotaMaremi/ginboard/utils"
	"github.com/HirotaMaremi/ginboard/valueObject"
	"github.com/gin-gonic/gin"
	"github.com/olahol/go-imageupload"
	"log"
	"os"
	"time"
)

type View struct {
	//Thread         model.Thread
	Comments       []model.Comment
	LikeCount      []model.LikeCount
	UserMap        map[int]string
	GoodCommentIds []uint
	LikeMap        map[uint]gin.H
	Pagination     valueObject.Page
	Pages          []int
}

func CreateThreadViewModel(thread model.Thread, c *gin.Context) (*View, error) {
	strCommentId := c.Query("comment")
	commentCount := len(thread.Comments)

	page := utils.ConvertPage(c, commentCount)
	comments, err := model.GetCommentPagination(page, thread.ID, strCommentId)
	likeCount, _ := model.GetLikePagination(page, thread.ID, strCommentId)
	if err != nil {
		log.Println(err.Error())
		return nil, errors.New("エラーが発生しました。")
	}

	//コメントしたユーザ名の取得
	userMap := make(map[int]string)
	for _, val := range comments {
		userMap[int(val.ID)] = model.GetOne(int(val.CreatedBy)).Name
	}
	//ページネーションのページ番号取得用
	pages := make([]int, page.TotalPages, page.TotalPages)
	for i := 0; i < page.TotalPages; i++ {
		pages[i] = i + 1
	}

	// いいね機能
	var goodCommentIds []uint
	likeMap := make(map[uint]gin.H)
	loginUserId, exist := c.Get("userID")
	if exist {
		idAndLikes, err := model.GetLikeCommentsByUserId(thread.ID, uint(loginUserId.(float64)))
		if err != nil {
			log.Println(err.Error())
			return nil, errors.New("エラーが発生しました。")
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

	viewModel := View{
		Comments:       comments,
		LikeCount:      likeCount,
		UserMap:        userMap,
		GoodCommentIds: goodCommentIds,
		LikeMap:        likeMap,
		Pagination:     page,
		Pages:          pages,
	}

	return &viewModel, nil
}

// 画像処理
func ProcessImg(c *gin.Context) *string {
	imageupload.LimitFileSize(1024*1024, c.Writer, c.Request)
	img, err := imageupload.Process(c.Request, "file")
	if err != nil {
		// 画像ファイルが設定されていない
		//c.Error(err).SetType(gin.ErrorTypePublic)
		return nil
	}
	thumb, err := imageupload.ThumbnailPNG(img, 150, 150)
	if err != nil {
		c.Error(err).SetType(gin.ErrorTypePublic)
		return nil
	}

	errDir := utils.CreateFolder("file")
	if errDir != nil {
		c.Error(errDir).SetType(gin.ErrorTypePublic)
		return nil
	}
	h := sha1.Sum(thumb.Data)
	filename := fmt.Sprintf("%s_%x.png",
		time.Now().Format("20060102150405"), h[:4])
	thumb.Save("file/" + filename)
	// S3Upload 5回までリトライ
	for i := 0; i < 5; i++ {
		err := UploadS3("file/"+filename, os.Getenv("ENV")+"/"+filename)
		if err == nil {
			os.Remove("file/" + filename)
			break
		}
	}

	return &filename
}

// コメント画像処理
func ProcessCommentImg(c *gin.Context, threadId uint) *string {
	imageupload.LimitFileSize(5*1024*1024, c.Writer, c.Request)
	img, err := imageupload.Process(c.Request, "file")
	if err != nil {
		// 画像ファイルが設定されていない
		//c.Error(err).SetType(gin.ErrorTypePublic)
		return nil
	}
	thumb, err := imageupload.ThumbnailPNG(img, 500, 500)
	if err != nil {
		c.Error(err).SetType(gin.ErrorTypePublic)
		return nil
	}

	dir := "file/comment/" + fmt.Sprint(threadId)
	errDir := utils.CreateFolder(dir)
	if errDir != nil {
		c.Error(errDir).SetType(gin.ErrorTypePublic)
		return nil
	}
	h := sha1.Sum(thumb.Data)
	filename := fmt.Sprintf("%s_%x.png",
		time.Now().Format("20060102150405"), h[:4])
	thumb.Save(dir + "/" + filename)
	// S3Upload 5回までリトライ
	for i := 0; i < 5; i++ {
		err := UploadS3(dir+"/"+filename, os.Getenv("ENV")+"/comment/"+fmt.Sprint(threadId)+"/"+filename)
		if err == nil {
			os.Remove(dir + "/" + filename)
			break
		}
		if err != nil && i == 4 {
			c.Error(err).SetType(gin.ErrorTypePublic)
			return nil
		}
	}

	return &filename
}
