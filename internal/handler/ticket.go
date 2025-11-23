package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type SeatItem struct {
	SeatNumber string `json:"seat_number"`
}

// Seats accepts either ["A1","A2"] or [{"seat_number":"A1"},...]
type Seats []SeatItem

func (s *Seats) UnmarshalJSON(data []byte) error {
	// try array of strings
	var arrStr []string
	if err := json.Unmarshal(data, &arrStr); err == nil {
		out := make(Seats, 0, len(arrStr))
		for _, v := range arrStr {
			out = append(out, SeatItem{SeatNumber: v})
		}
		*s = out
		return nil
	}

	// try array of objects
	var arrObj []SeatItem
	if err := json.Unmarshal(data, &arrObj); err == nil {
		*s = Seats(arrObj)
		return nil
	}

	return errors.New("invalid seats format")
}

type BookRequest struct {
	ScheduleID uint  `json:"schedule_id" binding:"required"`
	Seats      Seats `json:"seats" binding:"required"`
}

func BookTicket(c *gin.Context) {
	var req BookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// handle validation errors
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make(map[string]string)
			for _, fe := range ve {
				out[fe.Field()] = fe.Tag()
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": out})
			return
		}

		// handle common JSON unmarshal mistakes and return friendlier messages
		msg := err.Error()
		if strings.Contains(msg, "cannot unmarshal string into Go struct field") {
			// try to guess the field name from the error message
			// example: cannot unmarshal string into Go struct field BookRequest.seats of type []struct
			parts := strings.Split(msg, " ")
			field := "body"
			for i, p := range parts {
				if p == "field" && i+1 < len(parts) {
					// next token is like BookRequest.seats
					fld := parts[i+1]
					if idx := strings.Index(fld, "."); idx != -1 {
						field = strings.ToLower(strings.TrimPrefix(fld[idx+1:], ""))
					} else {
						field = strings.ToLower(fld)
					}
					break
				}
			}
			// provide a clearer message for seats specifically
			if strings.Contains(msg, ".seats") || field == "seats" {
				c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"seats": "must be an array of objects like [{\"seat_number\": \"A1\"}]"}})
				return
			}

			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format for field: " + field})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	var schedule model.Schedule
	if err := db.DB.First(&schedule, req.ScheduleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	uidVal, _ := c.Get("user_id")
	userID := uidVal.(uint)

	tx := model.Transaction{UserID: userID, ScheduleID: req.ScheduleID, TransactionTime: time.Now()}
	if err := db.DB.Create(&tx).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total := 0
	for _, s := range req.Seats {
		td := model.TransactionDetail{TransactionID: tx.ID, Seat: s.SeatNumber, Price: schedule.Price}
		db.DB.Create(&td)
		total += schedule.Price
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Tiket berhasil dipesan", "data": gin.H{"transaction_id": tx.ID, "total_amount": total, "seats": req.Seats, "schedule_id": req.ScheduleID}})
}

func GetMyBookings(c *gin.Context) {
	uidVal, _ := c.Get("user_id")
	userID := uidVal.(uint)
	var bookings []model.Transaction
	db.DB.Preload("Details").Where("user_id = ?", userID).Find(&bookings)
	out := make([]gin.H, 0, len(bookings))
	for _, b := range bookings {
		var schedule model.Schedule
		if err := db.DB.Preload("Movie.Genres").Preload("Theater").First(&schedule, b.ScheduleID).Error; err != nil {
			// skip if schedule not found
			continue
		}

		seats := make([]string, 0, len(b.Details))
		total := 0
		for _, d := range b.Details {
			seats = append(seats, d.Seat)
			total += d.Price
		}

		genres := make([]string, 0)
		if schedule.Movie != nil {
			for _, g := range schedule.Movie.Genres {
				genres = append(genres, g.Name)
			}
		}

		movieTitle := ""
		movieDuration := 0
		moviePoster := ""
		if schedule.Movie != nil {
			movieTitle = schedule.Movie.Title
			movieDuration = schedule.Movie.Duration
			moviePoster = schedule.Movie.Poster
		}

		theaterName := ""
		if schedule.Theater != nil {
			theaterName = schedule.Theater.Name
		}

		out = append(out, gin.H{
			"id":             b.ID,
			"movie_title":    movieTitle,
			"movie_duration": movieDuration,
			"movie_poster":   moviePoster,
			"movie_genre":    genres,
			"schedule_date":  schedule.StartTime,
			"theater_name":   theaterName,
			"seats":          seats,
			"total_price":    total,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bookings retrieved", "data": out})
}

func GetBookingByID(c *gin.Context) {
	bookingID := c.Param("id")

	var booking model.Transaction
	if err := db.DB.Preload("Details").First(&booking, bookingID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	}

	var schedule model.Schedule
	if err := db.DB.Preload("Movie.Genres").Preload("Theater").First(&schedule, booking.ScheduleID).Error; err != nil {
		// skip if schedule not found
	}

	seats := make([]string, 0, len(booking.Details))
	total := 0
	for _, d := range booking.Details {
		seats = append(seats, d.Seat)
		total += d.Price
	}

	genres := make([]string, 0)
	if schedule.Movie != nil {
		for _, g := range schedule.Movie.Genres {
			genres = append(genres, g.Name)
		}
	}

	movieTitle := ""
	movieDuration := 0
	moviePoster := ""
	if schedule.Movie != nil {
		movieTitle = schedule.Movie.Title
		movieDuration = schedule.Movie.Duration
		moviePoster = schedule.Movie.Poster
	}

	theaterName := ""
	if schedule.Theater != nil {
		theaterName = schedule.Theater.Name
	}

	response := gin.H{
		"id":             booking.ID,
		"movie_title":    movieTitle,
		"movie_duration": movieDuration,
		"movie_poster":   moviePoster,
		"movie_genre":    genres,
		"schedule_date":  schedule.StartTime,
		"theater_name":   theaterName,
		"seats":          seats,
		"total_price":    total,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Booking retrieved",
		"data":    response,
	})
}