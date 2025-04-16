package monitoring

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/xuri/excelize/v2"
)

// Monitoring struktura
type Monitoring struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Boshqaruv string    `json:"boshqaruv"`
	Value     int       `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

// BazaStruct
type BazaStruct struct {
	db *sql.DB
}

// RequestBody
type RequestBody struct {
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

// NewBazaStruct
func NewBazaStructMonitor(db *sql.DB) *BazaStruct {
	return &BazaStruct{db: db}
}

// getMonitoringByDateRange
func (b *BazaStruct) getMonitoringByDateRange(startDate, endDate string) ([]Monitoring, error) {
	// Sana va vaqtni aniq belgilash
	startDateTime := fmt.Sprintf("%s 00:00:00", startDate)
	endDateTime := fmt.Sprintf("%s 23:59:59.999999", endDate)

	query := `
		SELECT id, email, boshqaruv, value, created_at
		FROM monitoring
		WHERE deleted_at IS NULL
		AND created_at BETWEEN $1 AND $2
		ORDER BY created_at
	`

	rows, err := b.db.Query(query, startDateTime, endDateTime)
	if err != nil {
		return nil, fmt.Errorf("monitoring ma'lumotlarini olish xatosi: %v", err)
	}
	defer rows.Close()

	var monitoringList []Monitoring
	for rows.Next() {
		var m Monitoring
		if err := rows.Scan(&m.ID, &m.Email, &m.Boshqaruv, &m.Value, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan xatosi: %v", err)
		}
		monitoringList = append(monitoringList, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows xatosi: %v", err)
	}

	return monitoringList, nil
}

// exportMonitoringToExcel
func exportMonitoringToExcel(monitoringList []Monitoring) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Monitoring"
	f.SetSheetName("Sheet1", sheet)

	// Sarlavhalar
	headers := []string{"ID", "Email", "Boshqaruv", "Value", "CreatedAt"}
	for col, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+col)
		f.SetCellValue(sheet, cell, header)
	}

	// Ma'lumotlarni yozish
	for row, m := range monitoringList {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row+2), m.ID)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row+2), m.Email)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row+2), m.Boshqaruv)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row+2), m.Value)
		// CreatedAt ni DD/MM/YYYY HH:MM:SS formatida yozish
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row+2), m.CreatedAt.Format("02/01/2006 15:04:05"))
	}

	// Ustun kengligini sozlash
	f.SetColWidth(sheet, "A", "E", 25)

	// Sarlavhalarga format
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0EBF5"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "E1", style)

	return f, nil
}

// handleExportMonitoring
func (b *BazaStruct) HandleExportMonitoring(c *gin.Context) {
	var req RequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("So'rov parse xatosi:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Noto'g'ri so'rov: start_date va end_date kerak"})
		return
	}

	log.Printf("So'rov qabul qilindi: start_date=%s, end_date=%s", req.StartDate, req.EndDate)

	// Sana formatini tekshirish
	_, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		log.Println("start_date formati xatosi:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date formati noto'g'ri (YYYY-MM-DD)"})
		return
	}
	_, err = time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		log.Println("end_date formati xatosi:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date formati noto'g'ri (YYYY-MM-DD)"})
		return
	}

	// Ma'lumotlarni olish
	monitoringList, err := b.getMonitoringByDateRange(req.StartDate, req.EndDate)
	if err != nil {
		log.Println("Monitoring ma'lumotlarini olish xatosi:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ma'lumotlarni olishda xato"})
		return
	}

	log.Printf("Topilgan yozuvlar soni: %d", len(monitoringList))

	// Agar ma'lumotlar bo'lmasa
	if len(monitoringList) == 0 {
		log.Println("Ma'lumot topilmadi:", req.StartDate, "dan", req.EndDate, "gacha")
		c.JSON(http.StatusOK, gin.H{"message": "Berilgan sanalarda ma'lumot topilmadi"})
		return
	}

	// Excel fayl yaratish
	f, err := exportMonitoringToExcel(monitoringList)
	if err != nil {
		log.Println("Excel fayl yaratish xatosi:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Excel fayl yaratishda xato"})
		return
	}

	// Fayl nomini dinamik shakllantirish
	filename := fmt.Sprintf("monitoring_%s_to_%s.xlsx", strings.ReplaceAll(req.StartDate, "-", ""), strings.ReplaceAll(req.EndDate, "-", ""))

	// Excel faylni to'g'ri yuborish
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Expires", "0")

	// Faylni yozish
	if err := f.Write(c.Writer); err != nil {
		log.Println("Excel fayl yuborish xatosi:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fayl yuborishda xato"})
		return
	}
}

type CreateM struct {
	Email     string `json:"email"`
	Boshqaruv string `json:"boshqaruv"`
	Value     int    `json:"value"`
}

func (b *BazaStruct) CreateMonitoring(c *gin.Context) {
	var monitoring Monitoring
	if err := c.ShouldBindJSON(&monitoring); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Noto'g'ri so'rov"})
		return
	}

	query := `
		INSERT INTO monitoring (id, email, boshqaruv, value, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	id := uuid.New().String()
	newtime := time.Now()
	_, err := b.db.Exec(query, id, monitoring.Email, monitoring.Boshqaruv, monitoring.Value, newtime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ma'lumotlarni saqlashda xato"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
