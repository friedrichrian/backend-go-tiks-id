package handler

import (
	"fmt"
	"net/http"
	"time"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
)

// GetTransactionIndex handles GET /transactions
func GetTransactionIndex(c *gin.Context) {
	transactions, err := model.GetTransactionIndex(db.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// ChartData represents single chart item returned to frontend
type ChartData struct {
	Day     string `json:"day"`
	Revenue int    `json:"revenue"`
}

// GetChartData handles GET /transaction/chart
// Query param: time_range (expected values: "Minggu Ini", "Bulan Ini", "Tahun Ini")
func GetChartData(c *gin.Context) {
	tr := c.DefaultQuery("time_range", "week")
	var data []ChartData

	now := time.Now()
	loc := now.Location()

	// helper to sum revenue between start (inclusive) and end (exclusive)
	sumRevenue := func(start, end time.Time) (int, error) {
		var res struct {
			Revenue int64 `gorm:"column:revenue"`
		}
		err := db.DB.Table("transaction t").Select("COALESCE(SUM(td.price),0) as revenue").
			Joins("join transaction_detail td on td.transaction_id = t.id").
			Where("t.transaction_time >= ? AND t.transaction_time < ?", start, end).
			Scan(&res).Error
		if err != nil {
			return 0, err
		}
		return int(res.Revenue), nil
	}

	switch tr {
	case "week":
		// start from Monday of current week
		weekday := int(now.Weekday())
		// Go: Sunday=0, Monday=1 ... adjust so Monday=0
		daysToMonday := (weekday + 6) % 7
		startOfWeek := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -daysToMonday)
		for i := 0; i < 7; i++ {
			d := startOfWeek.AddDate(0, 0, i)
			next := d.AddDate(0, 0, 1)
			rev, err := sumRevenue(d, next)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to aggregate revenue"})
				return
			}
			dayNames := map[time.Weekday]string{
				time.Monday:    "Senin",
				time.Tuesday:   "Selasa",
				time.Wednesday: "Rabu",
				time.Thursday:  "Kamis",
				time.Friday:    "Jumat",
				time.Saturday:  "Sabtu",
				time.Sunday:    "Minggu",
			}
			data = append(data, ChartData{Day: dayNames[d.Weekday()], Revenue: rev})
		}
	case "month":
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end := first.AddDate(0, 1, 0)
		// create up to 4 week buckets (Minggu 1..4)
		for i := 0; i < 4; i++ {
			start := first.AddDate(0, 0, i*7)
			if start.Before(first) {
				start = first
			}
			finish := start.AddDate(0, 0, 7)
			if finish.After(end) {
				finish = end
			}
			// if start >= end then break
			if !start.Before(end) {
				break
			}
			rev, err := sumRevenue(start, finish)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to aggregate revenue"})
				return
			}
			label := fmt.Sprintf("Minggu %d", i+1)
			data = append(data, ChartData{Day: label, Revenue: rev})
		}
	case "year":
		year := now.Year()
		monthNames := []string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
		for m := 1; m <= 12; m++ {
			s := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, loc)
			e := s.AddDate(0, 1, 0)
			rev, err := sumRevenue(s, e)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to aggregate revenue"})
				return
			}
			data = append(data, ChartData{Day: monthNames[m-1], Revenue: rev})
		}
	default:
		data = []ChartData{}
	}

	c.JSON(http.StatusOK, gin.H{"message": "success", "data": data})
}
