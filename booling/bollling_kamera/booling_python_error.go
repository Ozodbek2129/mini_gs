package booling

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var file_name = "booling_python_error.json"

type BoolingPythonStruct struct {
	Key   string `json:"key"`
	Value bool   `json:"value"`
}

func readJSONFile_python() (map[string]bool, error) {
	file, err := ioutil.ReadFile(file_name)
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

func writeJSONFile_python(data map[string]bool) error {
	fileData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file_name, fileData, 0644)
}

func BoolingPostPython(c *gin.Context) {
	var booling BoolingPythonStruct
	if err := c.ShouldBindJSON(&booling); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	booling1, err := readJSONFile_python()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read JSON file"})
		return
	}

	if _, exists := booling1[booling.Key]; exists {
		booling1[booling.Key] = booling.Value
		err = writeJSONFile_python(booling1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update JSON file"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Value updated successfully"})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Key not found, no changes made"})
	}
}

func BoolingReadPython(c *gin.Context) {
	data, err := readJSONFile_python()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read JSON file"})
		return
	}

	c.JSON(http.StatusOK, data)
}

var clients1 = make(map[*websocket.Conn]bool)
var watcher1 *fsnotify.Watcher
var lastSentData11 []byte

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartFileWatcher_Python_Bool() {
	var err error
	watcher1, err = fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	// Faylni kuzatishga qo'shish
	err = watcher1.Add(file_name)
	if err != nil {
		log.Println("Faylni kuzatishga qo'shishda xato:", err)
	}

	go watchFileChanges()
}

func readJSONFile_Bool_Python() (map[string]bool, error) {
	file, err := ioutil.ReadFile(file_name)
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

func watchFileChanges() {
	for {
		select {
		case event, ok := <-watcher1.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("bool_python_error.json fayli o'zgardi, mijozlarga yuborilmoqda")

				updatedData, err := readJSONFile_Bool_Python()

				if err == nil {
					broadcastUpdate(updatedData)
				}
			}
		case err, ok := <-watcher1.Errors:
			if !ok {
				return
			}
			log.Println("Watcher xatosi:", err.Error())
		}
	}
}

func WebSocketHandler_Python_Bool(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket ulanishida xato:", err)
		return
	}
	clients1[conn] = true

	log.Println("Yangi mijoz ulandi, barcha ma'lumotlar yuborilmoqda")
	data, err := readJSONFile_Bool_Python()
	if err == nil {
		conn.WriteJSON(data)
	}

	go func() {
		defer func() {
			delete(clients1, conn)
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

func broadcastUpdate(data map[string]bool) {

	message, err := json.Marshal(data)
	if err != nil {
		return
	}

	if string(message) == string(lastSentData11) {
		return // Agar ma'lumot oldingi yuborilgan ma'lumot bilan bir xil bo'lsa, yuborilmaydi
	}

	lastSentData11 = message // Yangi ma'lumotni saqlab qo'yamiz

	for client := range clients1 {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			client.Close()
			delete(clients1, client)
		}
	}
}
