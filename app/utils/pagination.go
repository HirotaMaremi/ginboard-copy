package utils

import (
	"github.com/HirotaMaremi/ginboard/valueObject"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"math"
	"strconv"
)

func ConvertPage(context *gin.Context, totalElements int) valueObject.Page {
	page, _ := strconv.Atoi(context.Query("pg"))
	if page == 0 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(context.Query("size"))
	if pageSize == 0 {
		pageSize = 100
	}
	switch {
	case pageSize > totalElements:
		pageSize = totalElements
	case pageSize > 100:
		pageSize = 100
	case pageSize <= 0:
		if totalElements < 100 {
			pageSize = totalElements
		} else {
			pageSize = 100
		}
	}
	totalPages := 0
	if totalElements != 0 {
		totalPages = int(math.Ceil(float64(totalElements) / float64(pageSize)))
	}

	return valueObject.Page{Number: page, Size: pageSize, TotalElements: totalElements, TotalPages: totalPages}
}

// gorm scopes
// https://gorm.io/ja_JP/docs/scopes.html
func Paginate(page valueObject.Page) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page.Number - 1) * page.Size
		return db.Offset(offset).Limit(page.Size)
	}
}

//db.Scopes(Paginate(r)).Find(&users)
//db.Scopes(Paginate(r)).Find(&articles)
