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

// UploadMedia godoc
// @Summary Faylni yuklash
// @Description Bu endpoint foydalanuvchiga rasm (.jpg, .jpeg, .png) yoki video (.mp4) fayllarni yuklash imkonini beradi. Fayl MinIO serveriga saqlanadi va URL qaytariladi.
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Yuklanadigan fayl (faqat .jpg, .jpeg, .png yoki .mp4 formatlari qabul qilinadi)"
// @Success 200 {object} map[string]string "Fayl muvaffaqiyatli yuklandi" Example({"file_url": "http://minio:9000/images/abc123.jpg"})
// @Failure 400 {object} map[string]string "Noto'g'ri so'rov yoki fayl turi" Example({"error": "Yaroqsiz fayl turi"})
// @Failure 500 {object} map[string]string "Server xatosi" Example({"error": "MinIO bilan bog'lanishda xatolik: ..."})
// @Router /upload_image [post]
func UploadMedia(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Faylni olishda xatolik: " + err.Error(),
		})
		return
	}

	// Fayl kengaytmasini olish
	fileExt := filepath.Ext(file.Filename)

	// Yangi fayl nomi generatsiya qilish (UUID + kengaytma)
	newFile := uuid.NewString() + fileExt

	// Lokal katalog yaratish
	mediaDir := "./media"
	err = os.MkdirAll(mediaDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Serverda katalog yaratishda xatolik: " + err.Error(),
		})
		return
	}

	// Faylni lokal katalogga saqlash
	filePath := filepath.Join(mediaDir, newFile)
	err = c.SaveUploadedFile(file, filePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Faylni saqlashda xatolik: " + err.Error(),
		})
		return
	}

	// MinIO klientini sozlash
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

	// Bucket nomi va Content-Type ni aniqlash
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

	// Bucket mavjudligini tekshirish
	exists, err := minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "MinIO bilan bucketni tekshirishda xatolik: " + err.Error(),
		})
		return
	}

	// Bucketni yaratish (agar mavjud bo'lmasa)
	if !exists {
		err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Bucket yaratishda xatolik: " + err.Error(),
			})
			return
		}
	}

	// Ommaviy kirish siyosatini o'rnatish
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": "arn:aws:s3:::%s/*"
			}
		]
	}`
	policy = fmt.Sprintf(policy, bucketName)
	err = minioClient.SetBucketPolicy(context.Background(), bucketName, policy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Bucket siyosatini o'rnatishda xatolik: " + err.Error(),
		})
		return
	}

	// Faylni MinIO'ga yuklash
	_, err = minioClient.FPutObject(context.Background(), bucketName, newFile, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "MinIO ga faylni yuklashda xatolik: " + err.Error(),
		})
		return
	}

	// Tashqi manzilni aniqlash (muhit o'zgaruvchisidan olish)
	minioHost := os.Getenv("MINIO_HOST")
	if minioHost == "" {
		minioHost = "54.93.213.231:9000" // Default qiymat
	}
	objUrl := fmt.Sprintf("http://%s/%s/%s", minioHost, bucketName, newFile)

	// Muvaffaqiyatli javob qaytarish
	c.JSON(http.StatusOK, gin.H{
		"file_url": objUrl,
	})

	// Lokal faylni o'chirish (ixtiyoriy, agar kerak bo'lmasa)
	err = os.Remove(filePath)
	if err != nil {
		// Faqat log qilish, javobga ta'sir qilmaydi
		fmt.Printf("Lokal faylni o'chirishda xatolik: %v\n", err)
	}
}