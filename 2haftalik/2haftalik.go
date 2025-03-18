package haftalik2

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Haftalik2Struct struct {
	db      *sql.DB
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

func NewHaftalik2Struct(db *sql.DB) *Haftalik2Struct {
	return &Haftalik2Struct{
		db:      db,
		clients: make(map[*websocket.Conn]bool),
	}
}

type Haftalik2Repo struct {
	Date     string `json:"date"`
	Day      string `json:"day"`
	Quantity int    `json:"quantity"`
}

func (h *Haftalik2Struct) Haftalik2(c *gin.Context) {
	query_insert := `insert into haftalik2(id, date, day, quantity, created_at, update_at) values($1, $2, $3, $4, $5, $6)`
	query_update := `update haftalik2 set quantity = $1, update_at = $2 where date = $3 and deleted_at is null`
	query_select := `select date, day, quantity from haftalik2 where date = $1 and deleted_at is null`

	var data Haftalik2Repo

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "binding error"})
		return
	}

	id := uuid.New().String()
	created_at := time.Now()

	var datee, dayy string
	var quantity int
	err := h.db.QueryRow(query_select, data.Date).Scan(&datee, &dayy, &quantity)
	if err != nil {
		if err == sql.ErrNoRows {
			_, err := h.db.Exec(query_insert, id, data.Date, data.Day, data.Quantity, created_at, created_at)
			if err != nil {
				c.JSON(400, gin.H{"error": "insert error"})
				return
			}
			c.JSON(200, gin.H{"message": "insert successful"})
			return
		}
		c.JSON(400, gin.H{"error": "selection error"})
		return
	}

	_, err = h.db.Exec(query_update, data.Quantity, data.Date)
	if err != nil {
		c.JSON(400, gin.H{"error": "update error"})
		return
	}
	c.JSON(200, gin.H{"message": "update successful"})
}

// ----------------------------------------------------------------------------------------------------------------------------------

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Haftalik2Struct) WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket ulanishida xato:", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	log.Println("Yangi mijoz ulandi")
	h.sendUpdatedData(conn)

	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *Haftalik2Struct) sendUpdatedData(conn *websocket.Conn) {
	data, err := h.getHaftalikData()
	if err != nil {
		log.Println("Ma'lumotni olishda xato:", err)
		return
	}
	conn.WriteJSON(data)
}

func (h *Haftalik2Struct) BroadcastUpdate() {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := h.getHaftalikData()
	if err != nil {
		log.Println("Bazadan ma'lumot olishda xato:", err)
		return
	}

	message, err := json.Marshal(data)
	if err != nil {
		return
	}

	for client := range h.clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			client.Close()
			delete(h.clients, client)
		}
	}
}

func (h *Haftalik2Struct) getHaftalikData() ([]Haftalik2Repo, error) {
	query := `SELECT date, day, quantity FROM haftalik2 WHERE deleted_at IS NULL AND date >= NOW() - INTERVAL '14 days' ORDER BY date ASC`
	rows, err := h.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Haftalik2Repo
	for rows.Next() {
		var record Haftalik2Repo
		if err := rows.Scan(&record.Date, &record.Day, &record.Quantity); err != nil {
			return nil, err
		}
		results = append(results, record)
	}
	return results, nil
}

func (h *Haftalik2Struct) Get2Haftalik(c *gin.Context) {
	query := `SELECT date, day, quantity FROM haftalik2 WHERE deleted_at IS NULL AND date >= NOW() - INTERVAL '14 days' ORDER BY date ASC`
	rows, err := h.db.Query(query)
	if err != nil {
		fmt.Println(err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []Haftalik2Repo
	for rows.Next() {
		var record Haftalik2Repo
		if err := rows.Scan(&record.Date, &record.Day, &record.Quantity); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		results = append(results, record)
	}

	c.JSON(200, gin.H{"data": results})
}
