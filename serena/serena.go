package serena

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type SerenaStruct struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}

var filename = "serena.json"

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

func SerenaPost(c *gin.Context) {
	var data SerenaStruct
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	data1, err := readJSONFile()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read JSON file"})
		return
	}

	if _, exists := data1[data.Key]; exists {
		data1[data.Key] = data.Value
	} else {
		data1[data.Key] = data.Value
	}
	err = writeJSONFile(data1)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to write JSON file"})
		return
	}
	c.JSON(200, gin.H{"message": "Data updated successfully"})
}

func SerenaGet(c *gin.Context) {
	data1, err := readJSONFile()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read JSON file"})
		return
	}
	c.JSON(200, data1)
}

// ------------------------------------------------------------------------------
var clients = make(map[*websocket.Conn]bool)
var watcher *fsnotify.Watcher
var lastSentData []byte

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartFileWatcher_serena() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

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
				log.Println("serena.json fayli o'zgardi, mijozlarga yuborilmoqda")

				updatedData, err := readJSONFile()

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

func WebSocketHandler_serena(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket ulanishida xato:", err)
		return
	}
	clients[conn] = true

	log.Println("Yangi mijoz ulandi, barcha ma'lumotlar yuborilmoqda")
	data, err := readJSONFile()
	if err == nil {
		conn.WriteJSON(data)
	}

	go func() {
		defer func() {
			delete(clients, conn)
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

	message, err := json.Marshal(data)
	if err != nil {
		return
	}

	if string(message) == string(lastSentData) {
		return 
	}

	lastSentData = message

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}
