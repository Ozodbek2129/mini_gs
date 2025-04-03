package micro_gs_data_blok_read

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool) // micro_gs.json uchun
var clients1 = make(map[*websocket.Conn]bool) // micro_gs1.json uchun
var watcher *fsnotify.Watcher
var lastSentData []byte  // micro_gs.json uchun
var lastSentData1 []byte // micro_gs1.json uchun

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var fileName = "micro_gs.json"
var fileName1 = "micro_gs1.json"

func StartFileWatcher() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	// Ikkala faylni kuzatishga qo'shish
	err = watcher.Add(fileName)
	if err != nil {
		log.Println("Faylni kuzatishga qo'shishda xato (micro_gs.json):", err)
	}
	err = watcher.Add(fileName1)
	if err != nil {
		log.Println("Faylni kuzatishga qo'shishda xato (micro_gs1.json):", err)
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
				switch event.Name {
				case fileName:
					log.Println("micro_gs.json fayli o'zgardi, mijozlarga yuborilmoqda")
					updatedData, err := loadData()
					if err != nil {
						log.Println("Ma'lumot yuklashda xato (micro_gs.json):", err)
						continue
					}
					broadcastUpdate(updatedData)
				case fileName1:
					log.Println("micro_gs1.json fayli o'zgardi, mijozlarga yuborilmoqda")
					updatedData, err := loadData1()
					if err != nil {
						log.Println("Ma'lumot yuklashda xato (micro_gs1.json):", err)
						continue
					}
					broadcastUpdate1(updatedData)
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

	file, err := os.Open(fileName)
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

func MicroGsDataBlokRead1(c *gin.Context) {

	file, err := os.Open(fileName1)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	var data MicroGsDataBlokReadStruct1
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

type MicroGsDataBlokReadStruct map[string]map[string]interface{}
type MicroGsDataBlokReadStruct1 map[string]map[string]interface{}

func saveData(data MicroGsDataBlokReadStruct) error {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fileName, file, 0644)
}

func saveData1(data MicroGsDataBlokReadStruct1) error {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fileName1, file, 0644)
}

func loadData() (MicroGsDataBlokReadStruct, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return MicroGsDataBlokReadStruct{}, err
	}

	var data MicroGsDataBlokReadStruct
	if err := json.Unmarshal(file, &data); err != nil {
		return MicroGsDataBlokReadStruct{}, err
	}

	return data, nil
}

func loadData1() (MicroGsDataBlokReadStruct1, error) {
	file, err := os.ReadFile(fileName1)
	if err != nil {
		return MicroGsDataBlokReadStruct1{}, err
	}

	var data MicroGsDataBlokReadStruct1
	if err := json.Unmarshal(file, &data); err != nil {
		return MicroGsDataBlokReadStruct1{}, err
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

func MicroGsDataBlokPost1(c *gin.Context) {
	var request struct {
		Category string      `json:"category"`
		Key      string      `json:"key"`
		Value    interface{} `json:"value"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingData, err := loadData1()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load data"})
		return
	}

	if _, exists := existingData[request.Category]; !exists {
		existingData[request.Category] = make(map[string]interface{})
	}

	existingData[request.Category][request.Key] = request.Value

	if err := saveData1(existingData); err != nil {
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
	clients[conn] = true

	log.Println("Yangi mijoz ulandi (micro_gs.json), barcha ma'lumotlar yuborilmoqda")
	data, err := loadData()
	if err == nil {
		err = conn.WriteJSON(data)
		if err != nil {
			log.Println("Mijozga dastlabki ma'lumot yuborishda xato (micro_gs.json):", err)
		}
	} else {
		log.Println("Dastlabki ma'lumot yuklashda xato (micro_gs.json):", err)
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

func WebSocketHandler1(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket ulanishida xato:", err)
		return
	}
	clients1[conn] = true

	log.Println("Yangi mijoz ulandi (micro_gs1.json), barcha ma'lumotlar yuborilmoqda")
	data, err := loadData1()
	if err == nil {
		err = conn.WriteJSON(data)
		if err != nil {
			log.Println("Mijozga dastlabki ma'lumot yuborishda xato (micro_gs1.json):", err)
		}
	} else {
		log.Println("Dastlabki ma'lumot yuklashda xato (micro_gs1.json):", err)
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

func broadcastUpdate(data MicroGsDataBlokReadStruct) {
	message, err := json.Marshal(data)
	if err != nil {
		log.Println("JSON marshal xatosi (micro_gs.json):", err)
		return
	}

	if string(message) == string(lastSentData) {
		log.Println("Ma'lumot o'zgarmagan (micro_gs.json), yuborilmaydi")
		return
	}

	lastSentData = message

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("Mijozga yuborishda xato (micro_gs.json):", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func broadcastUpdate1(data MicroGsDataBlokReadStruct1) {
	message, err := json.Marshal(data)
	if err != nil {
		log.Println("JSON marshal xatosi (micro_gs1.json):", err)
		return
	}

	if string(message) == string(lastSentData1) {
		log.Println("Ma'lumot o'zgarmagan (micro_gs1.json), yuborilmaydi")
		return
	}

	lastSentData1 = message

	for client := range clients1 {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("Mijozga yuborishda xato (micro_gs1.json):", err)
			client.Close()
			delete(clients1, client)
		}
	}
}