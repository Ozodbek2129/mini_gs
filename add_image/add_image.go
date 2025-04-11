package add_image

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// @Summary Upload Photo
// @Description Api for upload a new photo
// @Tags auth_media
// @Accept multipart/form-data
// @Param        email  path      string  true  "User Email"
// @Param file formData file true "createUserModel"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /auth/products/media/{email} [post]
func UploadMedia(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Faylni olishda xatolik: " + err.Error(),
		})
		return
	}

	fileExt := filepath.Ext(file.Filename)

	newFile := uuid.NewString() + fileExt

	mediaDir := "./media"
	err = os.MkdirAll(mediaDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Serverda katalog yaratishda xatolik: " + err.Error(),
		})
		return
	}

	filePath := filepath.Join(mediaDir, newFile)
	err = c.SaveUploadedFile(file, filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Faylni saqlashda xatolik: " + err.Error(),
		})
		return
	}

	minioClient, err := minio.New("minio:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minio", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "MinIO bilan bog'lanishda xatolik: " + err.Error(),
		})
		return
	}

	var bucketName string
	contentType := "application/octet-stream"

	switch fileExt {
	case ".jpg", ".jpeg", ".png":
		bucketName = "images"
		if fileExt == ".jpg" || fileExt == ".jpeg" {
			contentType = "image/jpeg"
		} else if fileExt == ".png" {
			contentType = "image/png"
		}
	case ".mp4":
		bucketName = "videos"
		contentType = "video/mp4"
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Yaroqsiz fayl turi",
		})
		return
	}

	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "MinIO bilan bucketni tekshirishda xatolik: " + err.Error(),
		})
		return
	}

	if !exists {
		err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Bucket yaratishda xatolik: " + err.Error(),
			})
			return
		}
	}

	_, err = minioClient.FPutObject(context.Background(), bucketName, newFile, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "MinIO ga faylni yuklashda xatolik: " + err.Error(),
		})
		return
	}

	objUrl := fmt.Sprintf("http://minio:9000/%s/%s", bucketName, newFile)
	c.JSON(http.StatusOK, gin.H{
		"file_url": objUrl,
	})
}