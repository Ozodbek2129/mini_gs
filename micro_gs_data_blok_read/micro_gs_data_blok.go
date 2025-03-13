package micro_gs_data_blok_read

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var mu sync.Mutex
var clients = make(map[*websocket.Conn]bool)
var watcher *fsnotify.Watcher
var lastSentData []byte

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartFileWatcher() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	// Faylni kuzatishga qo'shish
	err = watcher.Add(file_name)
	if err != nil {
		log.Println("Faylni kuzatishga qo'shishda xato:", err)
	}

	go watchFileChanges()
}

func watchFileChanges() {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("micro_gs.json fayli o'zgardi, mijozlarga yuborilmoqda")
				
				mu.Lock()
				updatedData, err := loadData()
				mu.Unlock()

				if err == nil {
					broadcastUpdate(updatedData)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Watcher xatosi:", err.Error())
		}
	}
}

func MicroGsDataBlokRead(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	filename := "micro_gs.json"

	file, err := os.Open(filename)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	var data MicroGsDataBlokReadStruct
	bytevalue, err := io.ReadAll(file)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	err = json.Unmarshal(bytevalue, &data)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, data)
}

var file_name = "micro_gs.json"

type MicroGsDataBlokReadStruct map[string]map[string]interface{}

func saveData(data MicroGsDataBlokReadStruct) error {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file_name, file, 0644)
}

func loadData() (MicroGsDataBlokReadStruct, error) {
	file, err := os.ReadFile(file_name)
	if err != nil {
		return MicroGsDataBlokReadStruct{}, err
	}

	var data MicroGsDataBlokReadStruct
	if err := json.Unmarshal(file, &data); err != nil {
		return MicroGsDataBlokReadStruct{}, err
	}

	return data, nil
}

func MicroGsDataBlokPost(c *gin.Context) {
	var request struct {
		Category string      `json:"category"`
		Key      string      `json:"key"`
		Value    interface{} `json:"value"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingData, err := loadData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load data"})
		return
	}

	if _, exists := existingData[request.Category]; !exists {
		existingData[request.Category] = make(map[string]interface{})
	}

	existingData[request.Category][request.Key] = request.Value

	if err := saveData(existingData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data updated successfully"})
}

func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket ulanishida xato:", err)
		return
	}
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	log.Println("Yangi mijoz ulandi, barcha ma'lumotlar yuborilmoqda")
	data, err := loadData()
	if err == nil {
		conn.WriteJSON(data)
	}

	go func() {
		defer func() {
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
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

func broadcastUpdate(data MicroGsDataBlokReadStruct) {
	mu.Lock()
	defer mu.Unlock()

	message, err := json.Marshal(data)
	if err != nil {
		return
	}

	if string(message) == string(lastSentData) {
		return // Agar ma'lumot oldingi yuborilgan ma'lumot bilan bir xil bo'lsa, yuborilmaydi
	}

	lastSentData = message // Yangi ma'lumotni saqlab qo'yamiz

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}