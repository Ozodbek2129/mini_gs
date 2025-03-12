package micro_gs_data_blok_read

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

var mu sync.Mutex

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

	mu.Lock()
	defer mu.Unlock()

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
