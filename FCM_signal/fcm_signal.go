package fcmsignal

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

var (
	TokenLock sync.Mutex
	FcmClient *messaging.Client
)

type BazaFcmStruct struct {
	db *sql.DB
}

func NewBazaFcmStruct(db *sql.DB) *BazaFcmStruct {
	return &BazaFcmStruct{
		db:  db,
	}
}

func InitFirebase() {
	ctx := context.Background()
	opt := option.WithCredentialsFile("serviceAccountKey.json")
	app, err := firebase.NewApp(ctx, nil, opt)

	if err != nil {
		log.Fatalf("❌ Firebase init xatosi: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("❌ Messaging client xatosi: %v", err)
	}

	FcmClient = client

	fmt.Println("✅ Firebase tayyor.")
}

func (b *BazaFcmStruct) RegisterHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "❌ Noto'g'ri token"})
		return
	}

	TokenLock.Lock()

	query := `insert into fcm (
								id, fcmtoken, created_at, update_at
							) values (
							 	$1, $2, $3, $4
							)`

	id := uuid.NewString()
	newtime := time.Now()
	_, err := b.db.Exec(query, id, req.Token, newtime, newtime)
	if err != nil {
		c.JSON(400, gin.H{"error": "Bazaga tokenni saqlashda xatolik"})
		return
	}

	fmt.Println("✅ Token qabul qilindi:", req.Token)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// func (b *BazaFcmStruct) NotifyAllHandler(c *gin.Context) {
// 	ctx := context.Background()

// 	query := `select fcmtoken from fcm where deleted_at is null`

// 	fcmtokens, err := b.db.Query(query)
// 	if err != nil {
// 		c.JSON(400, gin.H{"error": "Bazadan fcm tokenni uqishda xatolik"})
// 		return
// 	}
// 	defer fcmtokens.Close()

// 	Tokens := make([]string, 0)

// 	for fcmtokens.Next() {
// 		var tokenss string
// 		if err := fcmtokens.Scan(&tokenss); err != nil {
// 			c.JSON(500, gin.H{"error": "FCM tokenni o'qib olishda xatolik"})
// 			return
// 		}

// 		Tokens = append(Tokens, tokenss)
// 	}

// 	if err := fcmtokens.Err(); err != nil {
// 		c.JSON(500, gin.H{"error": "Qatorlarni o'qishda xatolik"})
// 		return
// 	}

// 	for _, token := range Tokens {
// 		msg := &messaging.Message{
// 			Token: token,
// 			Notification: &messaging.Notification{
// 				Title: "Diqqat!",
// 				Body:  "Tizimda nosozlik aniqlandi!",
// 			},
// 			Data: map[string]string{
// 				"sound":  "signal",
// 				"action": "avariya",
// 			},
// 			Android: &messaging.AndroidConfig{
// 				Priority: "high",
// 				Notification: &messaging.AndroidNotification{
// 					Sound:     "signal",
// 					ChannelID: "alarm_channel",
// 				},
// 			},
// 		}

// 		resp, err := FcmClient.Send(ctx, msg)
// 		if err != nil {
// 			fmt.Println("❌ Xato:", err)
// 		} else {
// 			fmt.Println("✅ Yuborildi:", resp)
// 		}
// 	}

// 	// Javob qaytarish
// 	c.JSON(http.StatusOK, gin.H{"status": "sent"})
// }


func (b *BazaFcmStruct) NotifyAllHandler(c *gin.Context) {
	ctx := context.Background()

	// URL query parameterlarni olish
	title := c.DefaultQuery("title", "Diqqat!")               // default qiymat
	body := c.DefaultQuery("body", "Tizimda nosozlik aniqlandi!") // default qiymat
	action := c.DefaultQuery("action", "avariya")             // default qiymat

	query := `select fcmtoken from fcm where deleted_at is null`

	fcmtokens, err := b.db.Query(query)
	if err != nil {
		c.JSON(400, gin.H{"error": "Bazadan fcm tokenni uqishda xatolik"})
		return
	}
	defer fcmtokens.Close()

	Tokens := make([]string, 0)

	for fcmtokens.Next() {
		var token string
		if err := fcmtokens.Scan(&token); err != nil {
			c.JSON(500, gin.H{"error": "FCM tokenni o'qib olishda xatolik"})
			return
		}
		Tokens = append(Tokens, token)
	}

	if err := fcmtokens.Err(); err != nil {
		c.JSON(500, gin.H{"error": "Qatorlarni o'qishda xatolik"})
		return
	}

	for _, token := range Tokens {
		msg := &messaging.Message{
			Token: token,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: map[string]string{
				"sound":  "signal",
				"action": action,
			},
			Android: &messaging.AndroidConfig{
				Priority: "high",
				Notification: &messaging.AndroidNotification{
					Sound:     "signal",
					ChannelID: "alarm_channel",
				},
			},
		}

		resp, err := FcmClient.Send(ctx, msg)
		if err != nil {
			fmt.Println("❌ Xato:", err)
		} else {
			fmt.Println("✅ Yuborildi:", resp)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "sent",
		"title":  title,
		"body":   body,
		"action": action,
	})
}
