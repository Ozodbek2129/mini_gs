package micro_gs_data_blok_read

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func saveData(data MicroGsDataBlokReadStruct) error {
	file, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fileName, file, 0644)
}

// func saveData1(data MicroGsDataBlokReadStruct1) error {
// 	file, err := json.MarshalIndent(data, "", "  ")
// 	if err != nil {
// 		return err
// 	}
// 	return os.WriteFile(fileName1, file, 0644)
// }

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

// func MicroGsDataBlokRead1(c *gin.Context) {

// 	file, err := os.Open(fileName1)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": err.Error()})
// 		return
// 	}
// 	defer file.Close()

// 	var data MicroGsDataBlokReadStruct1
// 	bytevalue, err := io.ReadAll(file)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": err.Error()})
// 		return
// 	}

// 	err = json.Unmarshal(bytevalue, &data)
// 	if err != nil {
// 		c.JSON(500, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(200, data)
// }

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

// func MicroGsDataBlokPost1(c *gin.Context) {
// 	var request struct {
// 		Category string      `json:"category"`
// 		Key      string      `json:"key"`
// 		Value    interface{} `json:"value"`
// 	}

// 	if err := c.ShouldBindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	existingData, err := loadData1()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load data"})
// 		return
// 	}

// 	if _, exists := existingData[request.Category]; !exists {
// 		existingData[request.Category] = make(map[string]interface{})
// 	}

// 	existingData[request.Category][request.Key] = request.Value

// 	if err := saveData1(existingData); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Data updated successfully"})
// }

// --------------------------------------------------------------------------------------------------------------
var (
	clients      = make(map[*websocket.Conn]bool)
	lastSentData []byte
	fileName     = "micro_gs.json"
	clientsMu    sync.Mutex
	upgrader     = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	watcher      *fsnotify.Watcher
)

type MicroGsDataBlokReadStruct map[string]map[string]interface{}

func loadData() (MicroGsDataBlokReadStruct, error) {
	file, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	var data MicroGsDataBlokReadStruct
	err = json.Unmarshal(file, &data)
	return data, err
}

func broadcastUpdate(data MicroGsDataBlokReadStruct) {
	message, err := json.Marshal(data)
	if err != nil {
		log.Println("JSON marshal xatosi:", err)
		return
	}
	if string(message) == string(lastSentData) {
		log.Println("Ma'lumot o'zgarmagan, yuborilmaydi")
		return
	}
	lastSentData = message

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println("Mijozga yuborishda xato:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket ulanishida xato:", err)
		return
	}

	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	data, err := loadData()
	if err == nil {
		_ = conn.WriteJSON(data)
	}

	go func() {
		defer func() {
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func StartWatcherMicroGs() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	err = watcher.Add(fileName)
	if err != nil {
		log.Println("Fayl kuzatuvga qo'shilmadi:", err)
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("micro_gs.json fayli o'zgardi")
					data, err := loadData()
					if err == nil {
						broadcastUpdate(data)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher xatosi:", err)
			}
		}
	}()
}

// --------------------------------------------------------------------------------------------------------------
// var (
// 	clients1      = make(map[*websocket.Conn]bool)
// 	lastSentData1 []byte
// 	fileName1     = "micro_gs1.json"
// 	clientsMu1    sync.Mutex
// 	upgrader1     = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
// 	watcher1      *fsnotify.Watcher
// )

// type MicroGsDataBlokReadStruct1 map[string]map[string]interface{}

// func loadData1() (MicroGsDataBlokReadStruct1, error) {
// 	file, err := os.ReadFile(fileName1)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var data MicroGsDataBlokReadStruct1
// 	err = json.Unmarshal(file, &data)
// 	return data, err
// }

// func broadcastUpdate1(data MicroGsDataBlokReadStruct1) {
// 	message, err := json.Marshal(data)
// 	if err != nil {
// 		log.Println("JSON marshal xatosi:", err)
// 		return
// 	}
// 	if string(message) == string(lastSentData1) {
// 		log.Println("Ma'lumot o'zgarmagan, yuborilmaydi")
// 		return
// 	}
// 	lastSentData1 = message

// 	for client := range clients1 {
// 		err := client.WriteMessage(websocket.TextMessage, message)
// 		if err != nil {
// 			log.Println("Mijozga yuborishda xato:", err)
// 			client.Close()
// 			delete(clients1, client)
// 		}
// 	}
// }

// func WebSocketHandler1(c *gin.Context) {
// 	conn, err := upgrader1.Upgrade(c.Writer, c.Request, nil)
// 	if err != nil {
// 		log.Println("WebSocket ulanishida xato:", err)
// 		return
// 	}

// 	clientsMu1.Lock()
// 	if len(clients1) >= 10 {
// 		log.Println("Chegaradan oshdi, ulanish rad etildi")
// 		conn.Close()
// 		clientsMu1.Unlock()
// 		return
// 	}
// 	clients1[conn] = true
// 	clientsMu1.Unlock()

// 	data, err := loadData1()
// 	if err == nil {
// 		_ = conn.WriteJSON(data)
// 	}

// 	go func() {
// 		defer func() {
// 			clientsMu1.Lock()
// 			delete(clients1, conn)
// 			clientsMu1.Unlock()
// 			conn.Close()
// 		}()
// 		for {
// 			if _, _, err := conn.ReadMessage(); err != nil {
// 				break
// 			}
// 		}
// 	}()
// }

// func StartWatcherMicroGs1() {
// 	var err error
// 	watcher1, err = fsnotify.NewWatcher()
// 	if err != nil {
// 		panic(err)
// 	}
// 	err = watcher1.Add(fileName1)
// 	if err != nil {
// 		log.Println("Fayl kuzatuvga qo'shilmadi:", err)
// 	}
// 	go func() {
// 		for {
// 			select {
// 			case event, ok := <-watcher1.Events:
// 				if !ok {
// 					return
// 				}
// 				if event.Op&fsnotify.Write == fsnotify.Write {
// 					log.Println("micro_gs1.json fayli o'zgardi")
// 					data, err := loadData1()
// 					if err == nil {
// 						broadcastUpdate1(data)
// 					}
// 				}
// 			case err, ok := <-watcher1.Errors:
// 				if !ok {
// 					return
// 				}
// 				log.Println("Watcher xatosi:", err)
// 			}
// 		}
// 	}()
// }
