package baza

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"gs/config"
	"gs/email"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type RegisterRepo struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	Full_name string `json:"full_name"`
	Image     string `json:"image"`
}

type BazaStruct struct {
	db  *sql.DB
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
			queryUpdate := `UPDATE gs SET email = $1, image = $2, full_name = $3, update_at = $4, deleted_at = NULL WHERE email = $5`
			_, err = b.db.Exec(queryUpdate,
				registerData.Email,
				registerData.Image,
				registerData.Full_name,
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
                                    id, email, image, full_name, created_at, update_at
                                ) VALUES (
                                    $1, $2, $3, $4, $5, $6
                                )`

		if registerData.Image == "" {
			registerData.Image = "https://cdn.pixabay.com/photo/2015/10/05/22/37/blank-profile-picture-973460_1280.png"
		}

		_, err = b.db.Exec(queryInsert,
			registerData.Id,
			registerData.Email,
			registerData.Image,
			registerData.Full_name,
			newTime,
			newTime,
		)

		if err != nil {
			log.Printf("Database insertion error: %v", err)
			c.JSON(500, gin.H{"error": "Database error during insert"})
			return
		}
	}

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

	if !approval.Approve {
		_, err := b.rdb.Get(ctx, approval.Email).Result()
		if err == redis.Nil {
			c.JSON(400, gin.H{"error": "No registration data found for this email1"})
			return
		} else if err != nil {
			log.Printf("Redis error: %v", err)
			c.JSON(500, gin.H{"error": "Redis error"})
			return
		}

		err = b.rdb.Del(ctx, approval.Email).Err()
		if err != nil {
			log.Printf("Error deleting data from Redis: %v", err)
			c.JSON(500, gin.H{"error": "Redis error"})
			return
		}

		c.JSON(200, gin.H{"message": "Registration not approved, data deleted"})
		return
	}

	dataJson, err := b.rdb.Get(ctx, approval.Email).Result()
	if err == redis.Nil {
		c.JSON(400, gin.H{"error": "No registration data found for this email2"})
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

	err = b.rdb.SetEX(ctx, approval.Email, dataJson, 10*time.Minute).Err()
	if err != nil {
		log.Printf("Error saving approved data to Redis: %v", err)
		c.JSON(500, gin.H{"error": "Redis error"})
		return
	}

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

	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email va parol talab qilinadi"})
		return
	}

	fmt.Println(email)

	query := `select id, email, image, full_name, active from gs where email = $1 and deleted_at is null`

	var result struct {
		Id        string `json:"id"`
		Email     string `json:"email"`
		Full_name string `json:"full_name"`
		Active    bool   `json:"active"`
		Image     string `json:"image"`
	}

	err := b.db.QueryRow(query, email).Scan(
		&result.Id,
		&result.Email,
		&result.Image,
		&result.Full_name,
		&result.Active,
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

	query := `select id, email, image, full_name from gs where email = $1 and deleted_at is null`

	var result struct {
		Id         string `json:"id"`
		Email      string `json:"email"`
		Image      string `json:"image"`
		Full_name  string `json:"full_name"`
		Created_at string `json:"created_at"`
		Update_at  string `json:"update_at"`
		Deleted_at string `json:"deleted_at"`
	}

	err := b.db.QueryRow(query, data.Email).Scan(
		&result.Id,
		&result.Email,
		&result.Image,
		&result.Full_name,
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

type ActiveRepo struct {
	Id    string `json:"id"`
	Activ bool   `json:"active"`
}

func (b *BazaStruct) Active(c *gin.Context) {
	query_update := `update gs set active = $1 where id = $2 and deleted_at is null`
	var data ActiveRepo
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "selection email error"})
		return
	}
	_, err := b.db.Exec(query_update, data.Activ, data.Id)
	if err != nil {
		fmt.Println("err", err)
		c.JSON(400, gin.H{"Error deleting ": err})
		return
	}
	c.JSON(200, gin.H{"message": "successful"})
}

var clients = make(map[*websocket.Conn]bool)

type User struct {
	ID       string
	Email    string
	Image    string
	FullName string
	Active   bool
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (b *BazaStruct) HandleWebSocket(c *gin.Context) {
	if b == nil || clients == nil {
		log.Println("BazaStruct yoki clients map nil")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server xatosi"})
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket yangilash xatosi:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "WebSocket ulanish xatosi"})
		return
	}

	clients[ws] = true
	log.Println("Yangi WebSocket mijoz ulandi, jami mijozlar:", len(clients))

	users, err := b.getAllUsers()
	if err != nil {
		log.Println("Dastlabki foydalanuvchilarni olish xatosi:", err)
		delete(clients, ws)
		ws.Close()
		return
	}
	if err := ws.WriteJSON(users); err != nil {
		log.Println("Dastlabki ma'lumotlarni yuborish xatosi:", err)
		delete(clients, ws)
		ws.Close()
		return
	}

	for {
		if _, _, err := ws.NextReader(); err != nil {
			delete(clients, ws)
			log.Println("WebSocket mijoz uzildi, jami mijozlar:", len(clients))
			ws.Close()
			break
		}
	}
}

func (b *BazaStruct) getAllUsers() ([]User, error) {
	query := `
        SELECT id, email, image, full_name, active
        FROM gs
        WHERE deleted_at IS NULL
        ORDER BY id
    `

	rows, err := b.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Email, &user.Image, &user.FullName, &user.Active)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (b *BazaStruct) NotifyClients(users []User) {

	log.Println("Mijozlarga yangilanish yuborilmoqda, jami mijozlar:", len(clients))
	for client := range clients {
		err := client.WriteJSON(users)
		if err != nil {
			log.Println("Mijozga yangilanish yuborish xatosi:", err)
			client.Close()
			delete(clients, client)
		} else {
			log.Println("Mijozga ma'lumot muvaffaqiyatli yuborildi")
		}
	}
}

func (b *BazaStruct) WatchDatabase() {
	cfg := config.Load()

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DB_USER, cfg.DB_PASSWORD, cfg.DB_HOST, cfg.DB_PORT, cfg.DB_NAME)

	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Println("Listener xatosi:", err)
		} else {
			log.Println("Listener holati:", ev)
		}
	})

	err := listener.Listen("user_changes")
	if err != nil {
		log.Fatal("Kanalni tinglash xatosi:", err)
	}
	log.Println("user_changes kanalini tinglash boshlandi")

	for {
		select {
		case notification := <-listener.Notify:
			if notification != nil {
				log.Println("O'zgarish aniqlandi:", notification.Extra)
				users, err := b.getAllUsers()
				if err != nil {
					log.Println("Yangilangan foydalanuvchilarni olish xatosi:", err)
					continue
				}
				b.NotifyClients(users)
			}
		case <-time.After(60 * time.Second):
			if err := listener.Ping(); err != nil {
				log.Println("Listener ping xatosi:", err)
			} else {
				log.Println("Listener ping muvaffaqiyatli")
			}
		}
	}
}

func (b *BazaStruct) GetAll(c *gin.Context) {
	users, err := b.getAllUsers()
	if err != nil {
		log.Println("Error fetching users:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}
