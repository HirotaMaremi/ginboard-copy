package main

import (
	"errors"
	"fmt"
	"github.com/HirotaMaremi/ginboard/controller"
	"github.com/HirotaMaremi/ginboard/model"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	errTz := os.Setenv("TZ", "Asia/Tokyo")
	if errTz != nil {
		fmt.Printf("cannot set env TZ: ", errTz)
	}

	env := os.Getenv("ENV")
	err := godotenv.Load(".env." + env)

	if err != nil {
		fmt.Printf("cannot read .env file: %v", err)
	}

	_ = os.Mkdir("var", os.ModePerm)

	var file *os.File
	path := "var/gin-" + time.Now().Format("2006-01-02") + ".log"
	_, error := os.Stat(path)
	if !errors.Is(error, os.ErrNotExist) {
		// file exist
		log.Print("exist")
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			//panic(err)
			log.Println(err)
		}

		file = f
	} else {
		fmt.Println("not exist")

		f, _ := os.Create(path)
		file = f
	}

	gin.DefaultWriter = io.MultiWriter(os.Stdout)
	log.SetOutput(file)

	db := model.Db
	router := controller.GetRouter(db)
	router.Run(":3003")

	// 	router := gin.Default()
	//
	// 	router.GET("/", func(c *gin.Context) {
	// 		c.JSON(200, gin.H{
	// 			"message": "Hello World",
	// 		})
	// 	})
	//
	// 	router.Run(":3003")
}
