package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vnkhanh/survey-server/utils"
)

func UploadFile(c *gin.Context) {
	fmt.Println("===> Nhận request upload")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không nhận được file"})
		return
	}
	fmt.Println("===> Nhận file:", fileHeader.Filename)

	fileID := fmt.Sprintf("%d", time.Now().UnixNano())
	fmt.Println("===> Gọi UploadToSupabase...")

	publicURL, err := utils.UploadToSupabase(fileHeader, fileHeader.Filename, fileID, "", "")
	if err != nil {
		fmt.Println("===> Upload lỗi:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Println("===> Upload xong:", publicURL)
	c.JSON(http.StatusOK, gin.H{
		"message": "Upload thành công",
		"url":     publicURL,
	})
}
