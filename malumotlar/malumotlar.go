package malumotlar

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MalumotlarStruct struct {
	db *sql.DB
}

func NewMalumotlarRepo(DB *sql.DB) *MalumotlarStruct {
	return &MalumotlarStruct{
		db: DB,
	}
}

type MalumotlarStruct1 struct {
	Malumotlar_name  string `json:"malumotlar_name"`
	Malumotlar_value int    `json:"malumotlar_value"`
	Malumotlar_time  string `json:"malumotlar_time"`
}

func (m *MalumotlarStruct) MalumotlarPost(c *gin.Context) {
	query := `insert into malumotlar (
										id, malumotlar_name, malumotlar_value, date, timee, created_at, update_at
									) values (
									 	$1, $2, $3, $4, $5, $6, $7)`

	var data MalumotlarStruct1

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Malumotlarni uqishda xatolik"})
		return
	}

	currentTime := time.Now()

	id := uuid.NewString()
	newtime := time.Now()
	date := currentTime.Format("2006-01-02")
	parsed, err1 := time.Parse("2006-01-02 15:04:05", data.Malumotlar_time)
	if err1 != nil {
		fmt.Println("format xato")
	}
	timee := parsed.Format("15:04:05")

	_, err := m.db.Exec(query, id, data.Malumotlar_name, data.Malumotlar_value, date, timee, newtime, newtime)
	if err != nil {
		c.JSON(400, gin.H{"error": "Malumotlarni bazaga kiritishda xatolik"})
		return
	}

	c.JSON(200, gin.H{"ok": "Malumotlar saqlandi"})
}

type result struct {
	Id              string    `json:"id"`
	MalumotlarName  string    `json:"malumotlar_name"`
	MalumotlarValue int       `json:"malumotlar_value"`
	Date            time.Time `json:"date"`
	Timee           time.Time `json:"time"`
	CreatedAt       time.Time `json:"created_at"`
	UpdateAt        time.Time `json:"update_at"`
}

func (m *MalumotlarStruct) MalumotlarGet(c *gin.Context) {
	malumotnomi := c.Query("malumotlar_name")
	malumotsana := c.Query("date")

	query := `SELECT id, malumotlar_name, malumotlar_value, date, timee, created_at, update_at 
              FROM malumotlar 
              WHERE deleted_at IS NULL`

	var rows *sql.Rows
	var err error
	var args []interface{}

	if malumotnomi != "" && malumotsana == "" {
		query += " AND malumotlar_name ILIKE $1"
		rows, err = m.db.Query(query, "%"+malumotnomi+"%")
	} else if malumotsana != "" && malumotnomi == "" {
		parsedDate, err := time.Parse("2006-01-02", malumotsana)
		if err != nil {
			c.JSON(400, gin.H{"error": "Noto'g'ri sana formati", "details": err.Error()})
			return
		}

		query += " AND date = $1"
		args = append(args, parsedDate)
		rows, err = m.db.Query(query, args...)
	} else if malumotnomi != "" && malumotsana != "" {
		parsedDate, err := time.Parse("2006-01-02", malumotsana)
		if err != nil {
			c.JSON(400, gin.H{"error": "Noto'g'ri sana formati", "details": err.Error()})
			return
		}
		query += " AND malumotlar_name ILIKE $1 AND date = $2"
		args = append(args, "%"+malumotnomi+"%", parsedDate)
		rows, err = m.db.Query(query, args...)
	} else {
		rows, err = m.db.Query(query)
	}

	if err != nil {
		c.JSON(400, gin.H{"error": "Bazadan ma'lumotlarni olishda xatolik", "details": err.Error()})
		return
	}
	defer rows.Close()

	var datas []result
	for rows.Next() {
		var data result
		err := rows.Scan(&data.Id, &data.MalumotlarName, &data.MalumotlarValue, &data.Date, &data.Timee, &data.CreatedAt, &data.UpdateAt)
		if err != nil {
			c.JSON(400, gin.H{"error": "Ma'lumotlarni skan qilishda xatolik", "details": err.Error()})
			return
		}
		datas = append(datas, data)
	}

	if err := rows.Err(); err != nil {
		c.JSON(400, gin.H{"error": "Rows da xatolik", "details": err.Error()})
		return
	}

	type jsonResult struct {
		Id              string `json:"id"`
		MalumotlarName  string `json:"malumotlar_name"`
		MalumotlarValue int    `json:"malumotlar_value"`
		Date            string `json:"date"`
		Timee           string `json:"time"`
		CreatedAt       string `json:"created_at"`
		UpdateAt        string `json:"update_at"`
	}

	var jsonDatas []jsonResult
	for _, data := range datas {
		jsonDatas = append(jsonDatas, jsonResult{
			Id:              data.Id,
			MalumotlarName:  data.MalumotlarName,
			MalumotlarValue: data.MalumotlarValue,
			Date:            data.Date.Format("2006-01-02"),
			Timee:           data.Timee.Format("15:04:05"),
			CreatedAt:       data.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdateAt:        data.UpdateAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(200, jsonDatas)
}
