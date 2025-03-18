package baza

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gs/email"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type RegisterRepo struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

type BazaStruct struct {
	db *sql.DB
	rdb *redis.Client
}

func NewBazaStruct(db *sql.DB, rdb *redis.Client) *BazaStruct {
	return &BazaStruct{
		db:  db,
		rdb: rdb,
	}
}

func (b *BazaStruct) Register(c *gin.Context) {
	var data RegisterRepo

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input data"})
		return
	}

	ctx := context.Background()

	dataJson, err := b.rdb.Get(ctx, data.Email).Result()
	if err == nil {
		log.Printf("Existing registration found for email: %s. Overwriting with new data.", data.Email)
	} else if err != redis.Nil {
		log.Printf("Error checking Redis for existing data: %v", err)
		c.JSON(500, gin.H{"error": "Redis error"})
		return
	}

	if err == nil {
		var existingData RegisterRepo
		if json.Unmarshal([]byte(dataJson), &existingData) == nil {
			data.Id = existingData.Id
		} else {
			data.Id = uuid.NewString()
		}
	} else {
		data.Id = uuid.NewString()
	}
	updatedDataJson, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error encoding JSON: %v", err)
		c.JSON(500, gin.H{"error": "Error encoding JSON"})
		return
	}

	err = b.rdb.SetEX(ctx, data.Email, updatedDataJson, 10*time.Minute).Err()
	if err != nil {
		log.Printf("Error saving to Redis: %v", err)
		c.JSON(500, gin.H{"error": "Redis error"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Registration data saved successfully",
		"id":      data.Id,
	})
}

type Confirmation struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

func (b *BazaStruct) ConfirmationRegister(c *gin.Context) {
	var data Confirmation

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input data"})
		return
	}

	ctx := context.Background()
	approvedKey := data.Email
	dataJson, err := b.rdb.Get(ctx, approvedKey).Result()
	if err == redis.Nil {
		c.JSON(400, gin.H{"error": "No approved registration data found for this email"})
		return
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		c.JSON(500, gin.H{"error": "Redis error"})
		return
	}

	var registerData RegisterRepo
	err = json.Unmarshal([]byte(dataJson), &registerData)
	if err != nil {
		log.Printf("Error decoding Redis data: %v", err)
		c.JSON(500, gin.H{"error": "Error processing data"})
		return
	}

	if registerData.Id != data.Id {
		c.JSON(400, gin.H{"error": "Invalid confirmation id"})
		return
	}

	newTime := time.Now()

	// Check if user already exists in the database
	var existingId string
	var deletedAt sql.NullTime
	queryCheck := `SELECT id, deleted_at FROM gs WHERE email = $1`
	err = b.db.QueryRow(queryCheck, registerData.Email).Scan(&existingId, &deletedAt)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Database query error: %v", err)
		c.JSON(500, gin.H{"error": "Database error during check"})
		return
	}

	if existingId != "" {
		if deletedAt.Valid {
			queryUpdate := `UPDATE gs SET email = $1 update_at = $2, deleted_at = NULL WHERE email = $3`
			_, err = b.db.Exec(queryUpdate,
				registerData.Email,
				newTime,
				registerData.Email,
			)
			if err != nil {
				log.Printf("Database update error: %v", err)
				c.JSON(500, gin.H{"error": "Database error during update"})
				return
			}
		} else {
			c.JSON(400, gin.H{"error": "Email is already active"})
			return
		}
	} else {
		queryInsert := `INSERT INTO gs (
                                    id, email, created_at, update_at
                                ) VALUES (
                                    $1, $2, $3, $4
                                )`

		_, err = b.db.Exec(queryInsert,
			registerData.Id,
			registerData.Email,
			newTime,
			newTime,
		)

		if err != nil {
			log.Printf("Database insertion error: %v", err)
			c.JSON(500, gin.H{"error": "Database error during insert"})
			return
		}
	}

	// Remove Redis key
	err = b.rdb.Del(ctx, approvedKey).Err()
	if err != nil {
		log.Printf("Error deleting Redis data: %v", err)
		c.JSON(500, gin.H{"error": "Error deleting Redis data"})
		return
	}

	c.JSON(200, gin.H{"message": "Registration confirmed and data updated successfully"})
}

func (b *BazaStruct) AdminApprove(c *gin.Context) {
	type AdminApproval struct {
		Email   string `json:"email"`
		Approve bool   `json:"approve"`
	}

	var approval AdminApproval
	if err := c.ShouldBindJSON(&approval); err != nil {
		c.JSON(400, gin.H{"error": "Invalid input data"})
		return
	}

	ctx := context.Background()

	// Agar tasdiqlash false bo‘lsa
	if !approval.Approve {
		// Redis'dan ma'lumotni o‘chirish
		_, err := b.rdb.Get(ctx, approval.Email).Result()
		if err == redis.Nil {
			c.JSON(400, gin.H{"error": "No registration data found for this email"})
			return
		} else if err != nil {
			log.Printf("Redis error: %v", err)
			c.JSON(500, gin.H{"error": "Redis error"})
			return
		}

		// Redis'dan o‘chirish
		err = b.rdb.Del(ctx, approval.Email).Err()
		if err != nil {
			log.Printf("Error deleting data from Redis: %v", err)
			c.JSON(500, gin.H{"error": "Redis error"})
			return
		}

		c.JSON(200, gin.H{"message": "Registration not approved, data deleted"})
		return
	}

	// Agar tasdiqlash true bo‘lsa
	dataJson, err := b.rdb.Get(ctx, approval.Email).Result()
	if err == redis.Nil {
		c.JSON(400, gin.H{"error": "No registration data found for this email"})
		return
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		c.JSON(500, gin.H{"error": "Redis error"})
		return
	}

	// Redis ma'lumotni JSON ga parse qilish
	var registerData RegisterRepo
	err = json.Unmarshal([]byte(dataJson), &registerData)
	if err != nil {
		log.Printf("Error decoding Redis data: %v", err)
		c.JSON(500, gin.H{"error": "Error processing data"})
		return
	}

	// Tasdiqlangan ma'lumotlarni Redis'ga vaqtinchalik saqlash
	err = b.rdb.SetEX(ctx, approval.Email, dataJson, 10*time.Minute).Err()
	if err != nil {
		log.Printf("Error saving approved data to Redis: %v", err)
		c.JSON(500, gin.H{"error": "Redis error"})
		return
	}

	// Email jo‘natish
	err = email.SendCode(registerData.Email, registerData.Id)
	if err != nil {
		log.Printf("Error sending email: %v", err)
		c.JSON(500, gin.H{"error": "Error sending email"})
		return
	}

	c.JSON(200, gin.H{"message": "User approved and email sent"})
}

type LoginRepo struct {
	Email string `json:"email"`
}

func (b *BazaStruct) Login(c *gin.Context) {
	email := c.Query("email")

	if email == ""  {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email va parol talab qilinadi"})
		return
	}

	fmt.Println(email)

	query := `select id, email from gs where email = $1 and deleted_at is null`

	var result struct {
		Id    string `json:"id"`
		Email string `json:"email"`
	}

	err := b.db.QueryRow(query, email).Scan(
		&result.Id,
		&result.Email,
	)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Foydalanuvchi topilmadi"})
			return
		}
		log.Printf("Ma'lumotni olishda xatolik: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ichki server xatosi"})
		return
	}

	access := false
	if result.Email == "asrorfaxriddinov10@gmail.com" {
		access = true
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   result,
		"access": access,
	})
}

type DeleteRepo struct {
	Id string `json:"id"`
}

func (b *BazaStruct) Delete(c *gin.Context) {
	var data DeleteRepo

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Println("err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "deletion error"})
		return
	}
	query := `update gs set deleted_at = $1 where id = $2 and deleted_at is null`

	_, err := b.db.Exec(query, time.Now(), data.Id)
	if err != nil {
		fmt.Println("err", err)
		c.JSON(400, gin.H{"Error deleting ": err})
		return
	}

	c.JSON(200, gin.H{"message": "successful"})
}

type EmailRepo struct {
	Email string `json:"email"`
}

func (b *BazaStruct) GetEmail(c *gin.Context) {
	var data EmailRepo

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "selection email error"})
		return
	}
	fmt.Println(data)

	query := `select id, email from gs where email = $1 and deleted_at is null`

	var result struct {
		Id          string `json:"id"`
		Email       string `json:"email"`
		Created_at  string `json:"created_at"`
		Update_at   string `json:"update_at"`
		Deleted_at  string `json:"deleted_at"`
	}

	err := b.db.QueryRow(query, data.Email).Scan(
		&result.Id,
		&result.Email,
	)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		log.Printf("Ma'lumotni olishda xatolik: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}