package dataspost

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var mu sync.Mutex

type DatasStruct struct {
	Key   string `json:"key"`
	Value int64 `json:"value"`
}

var filename = "datas.json"

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