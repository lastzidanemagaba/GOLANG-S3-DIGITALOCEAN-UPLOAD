package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxFileSize = 200 * 1024 * 1024 // 200MB

func main() {
	key := ""
	secret := ""
	endpointURL := ""
	region := ""
	bucketName := ""
	folderName := ""

	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:         aws.String(endpointURL),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(false),
	}
	newSession, err := session.NewSession(s3Config)
	if err != nil {
		panic(err)
	}
	s3Client := s3.New(newSession)

	r := gin.Default()
	r.POST("/upload", func(c *gin.Context) {
		photo, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if photo.Size > maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File size too large"})
			return
		}

		fileExt := filepath.Ext(photo.Filename)
		contentType, err := detectContentType(photo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		randomName := uuid.New().String()
		objectName := folderName + "/" + randomName + fileExt
		file, err := photo.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(objectName),
			Body:        bytes.NewReader(data),
			ACL:         aws.String("public-read"),
			ContentType: aws.String(contentType),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		publicURL := fmt.Sprintf("%s/%s/%s", endpointURL, bucketName, objectName)
		c.JSON(http.StatusOK, gin.H{
			"message":        "File uploaded and publicly accessible",
			"url":            publicURL,
			"content_type":   contentType,
			"file_extension": fileExt,
		})
	})

	r.Run(":4342")
}

func detectContentType(photo *multipart.FileHeader) (string, error) {
	file, err := photo.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	mime := mimetype.Detect(data)
	return mime.String(), nil
}
