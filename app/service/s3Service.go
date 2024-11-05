package service

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"log"
	"os"
)

// S3のバケット名
var bucket = "ginboard"

func UploadS3(filePath string, key string) error {
	// 画像を読み込みます
	imageFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
		return err
	}
	// 最後に画像ファイルを閉じます
	defer imageFile.Close()

	// sessionを作成します
	newSession := session.Must(session.NewSession())

	awsRegion := os.Getenv("AWS_DEFAULT_REGION")
	// S3クライアントを作成します
	svc := s3.New(newSession,
		aws.NewConfig().WithRegion(awsRegion),
	)

	// S3にアップロードする内容をparamsに入れます
	params := &s3.PutObjectInput{
		// Bucket アップロード先のS3のバケット
		Bucket: aws.String(bucket),
		// Key アップロードする際のオブジェクト名
		Key: aws.String(key),
		// Body アップロードする画像ファイル
		Body: imageFile,
	}

	// S3にアップロードします
	_, err = svc.PutObject(params)
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Println("S3へアップロードが完了しました。")
	return nil
}

