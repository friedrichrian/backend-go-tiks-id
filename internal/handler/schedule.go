package handler

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func ScheduleIndex(c *gin.Context) {
	var schedules []model.Schedule
	db.DB.Preload("Movie").Preload("Theater").Find(&schedules)
	if len(schedules) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Schedule not found"})
		return
	}
	data := []gin.H{}
	for _, s := range schedules {
		data = append(data, gin.H{
			"id":                  s.ID,
			"movie_name":          s.Movie.Title,
			"movie_poster":        formatPosterURL(c, s.Movie.Poster),
			"theater_name":        s.Theater.Name,
			"movie_duration":      s.Movie.Duration,
			"schedule_price":      s.Price,
			"schedule_start_time": s.StartTime,
		})
	}
	c.JSON(http.StatusOK, gin.H{"message": "Schedule found", "data": data})
}

// ScheduleCreate validates input and ensures a theater is not already booked
// for the same start_time that hasn't passed yet. If the existing schedule's
// start_time is in the past, creating a new schedule at that time is allowed.
func ScheduleCreate(c *gin.Context) {
	var payload struct {
		MovieID   uint   `json:"movie_id" binding:"required"`
		TheaterID uint   `json:"theater_id" binding:"required"`
		StartTime string `json:"start_time" binding:"required"`
		Price     int    `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make(map[string]string)
			for _, fe := range ve {
				out[fe.Field()] = fe.Tag()
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": out})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// parse start time
	layout := "2006-01-02 15:04:05"
	if _, err := time.Parse(layout, payload.StartTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"StartTime": "invalid datetime format, expected '2006-01-02 15:04:05'"}})
		return
	}

	// check existing schedules for same theater and same start_time
	var existing []model.Schedule
	if err := db.DB.Where("theater_id = ? AND start_time = ?", payload.TheaterID, payload.StartTime).Find(&existing).Error; err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	for _, ex := range existing {
		// parse existing start time
		exT, perr := time.Parse(layout, ex.StartTime)
		if perr != nil {
			// if existing has bad format, be conservative and block
			c.JSON(http.StatusConflict, gin.H{"message": "Theater is already booked for that time"})
			return
		}
		// if existing schedule time is now or future, block
		if !exT.Before(now) {
			c.JSON(http.StatusConflict, gin.H{"message": "Theater is already booked for that time"})
			return
		}
	}

	s := model.Schedule{
		MovieID:   payload.MovieID,
		TheaterID: payload.TheaterID,
		StartTime: payload.StartTime,
		Price:     payload.Price,
	}
	if err := db.DB.Create(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Schedule created successfully", "data": s})
}

func ScheduleEdit(c *gin.Context) {
	id := c.Param("id")
	var s model.Schedule
	if err := db.DB.First(&s, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Schedule not found"})
		return
	}

	var payload struct {
		MovieID   uint   `json:"movie_id" binding:"required"`
		TheaterID uint   `json:"theater_id" binding:"required"`
		StartTime string `json:"start_time" binding:"required"`
		Price     int    `json:"price" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make(map[string]string)
			for _, fe := range ve {
				out[fe.Field()] = fe.Tag()
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": out})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// parse start time
	layout := "2006-01-02 15:04:05"
	if _, err := time.Parse(layout, payload.StartTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"StartTime": "invalid datetime format, expected '2006-01-02 15:04:05'"}})
		return
	}

	// check existing schedules for same theater and same start_time excluding current schedule
	var existing []model.Schedule
	if err := db.DB.Where("theater_id = ? AND start_time = ? AND id <> ?", payload.TheaterID, payload.StartTime, s.ID).Find(&existing).Error; err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	for _, ex := range existing {
		exT, perr := time.Parse(layout, ex.StartTime)
		if perr != nil {
			c.JSON(http.StatusConflict, gin.H{"message": "Theater is already booked for that time"})
			return
		}
		if !exT.Before(now) {
			c.JSON(http.StatusConflict, gin.H{"message": "Theater is already booked for that time"})
			return
		}
	}

	s.MovieID = payload.MovieID
	s.TheaterID = payload.TheaterID
	s.StartTime = payload.StartTime
	s.Price = payload.Price
	if err := db.DB.Save(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated successfully", "data": s})
}

func ScheduleDelete(c *gin.Context) {
	id := c.Param("id")
	var s model.Schedule
	if err := db.DB.First(&s, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Schedule not found"})
		return
	}
	db.DB.Delete(&s)
	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}
