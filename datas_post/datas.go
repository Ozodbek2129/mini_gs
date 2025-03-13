package dataspost

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/fsnotify/fsnotify"
)

var mu sync.Mutex

type DatasStruct struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

var filename = "datas.json"
var DatasStruct_ws map[string]int64

func readJSONFile() (map[string]int64, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var data map[string]int64
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func writeJSONFile(data map[string]int64) error {
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, fileData, 0644)
}

func DatasPost(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	var data DatasStruct

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println(data)

	data1, err := readJSONFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read JSON file"})
		return
	}

	if _, exists := data1[data.Key]; exists {
		data1[data.Key] = data.Value
		err = writeJSONFile(data1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update JSON file"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Value updated successfully"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Key not found, no changes made"})
	}
}

func DatasRead(c *gin.Context) {
	mu.Lock()
	defer mu.Unlock()

	data, err := readJSONFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read JSON file"})
		return
	}

	c.JSON(http.StatusOK, data)
}

var clients = make(map[*websocket.Conn]bool)
var watcher *fsnotify.Watcher
var lastSentData []byte

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartFileWatcher_datas() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	// Faylni kuzatishga qo'shish
	err = watcher.Add(filename)
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
				log.Println("datas.json fayli o'zgardi, mijozlarga yuborilmoqda")

				mu.Lock()
				updatedData, err := readJSONFile()
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

func WebSocketHandler_datas(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket ulanishida xato:", err)
		return
	}
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	log.Println("Yangi mijoz ulandi, barcha ma'lumotlar yuborilmoqda")
	data, err := readJSONFile()
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

func broadcastUpdate(data map[string]int64) {
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
