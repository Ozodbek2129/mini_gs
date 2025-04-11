package booling_kamera

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var filename = "booling.json"

type BoolingStruct struct {
	Key   string `json:"key"`
	Value bool   `json:"value"`
}

func readJSONFile() (map[string]bool, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var data map[string]bool
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func writeJSONFile(data map[string]bool) error {
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, fileData, 0644)
}

func BoolingPost(c *gin.Context) {
	var booling BoolingStruct
	if err := c.ShouldBindJSON(&booling); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	booling1, err := readJSONFile()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read JSON file"})
		return
	}

	if _, exists := booling1[booling.Key]; exists {
		booling1[booling.Key] = booling.Value
		err = writeJSONFile(booling1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update JSON file"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Value updated successfully"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Key not found, no changes made"})
	}
}

func BoolingRead(c *gin.Context) {
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

var upgrader_kamera = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartFileWatcher_Python_kamera() {
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

	go watchFileChanges_kamera()
}

func watchFileChanges_kamera() {
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
					broadcastUpdate_kamera(updatedData)
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

func WebSocketHandler_Python_kamera(c *gin.Context) {
	conn, err := upgrader_kamera.Upgrade(c.Writer, c.Request, nil)
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

func broadcastUpdate_kamera(data map[string]bool) {

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
