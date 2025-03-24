package booling

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
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