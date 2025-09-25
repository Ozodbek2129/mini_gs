package python_error

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var filename = "python_error.json"

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

func Python_error(c *gin.Context) {
	var request struct {
		Key   string `json:"key"`
		Value int    `json:"value"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	data, err := readJSONFile()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read JSON file"})
		return
	}

	if _, exists := data[request.Key]; exists {
		data[request.Key] = int64(request.Value)
		err = writeJSONFile(data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update JSON file"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Value updated successfully"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Key not found, no changes made"})
	}
}

func Python_error_read(c *gin.Context) {
	data, err := readJSONFile()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read JSON file"})
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

func StartFileWatcher_Python() {
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
				log.Println("python_error.json fayli o'zgardi, mijozlarga yuborilmoqda")

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

func WebSocketHandler_Python(c *gin.Context) {
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
