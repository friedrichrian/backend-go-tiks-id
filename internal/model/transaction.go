package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	ID              uint                `gorm:"primaryKey" json:"id"`
	UserID          uint                `json:"user_id"`
	User            User                `json:"user" gorm:"foreignKey:UserID"`
	ScheduleID      uint                `json:"schedule_id"`
	Schedule        Schedule            `json:"schedule" gorm:"foreignKey:ScheduleID"`
	TransactionTime time.Time           `json:"transaction_time"`
	Details         []TransactionDetail `json:"transaction_details" gorm:"foreignKey:TransactionID"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

type TransactionResponse struct {
	Message string          `json:"message"`
	Data    TransactionData `json:"data"`
}

type TransactionData struct {
	TotalTransactions int               `json:"total_transaction"`
	TicketsSold       int               `json:"tickets_sold"`
	TotalRevenue      int               `json:"total_revenue"`
	Transactions      []TransactionItem `json:"transaction"`
}

type TransactionItem struct {
	ID      uint   `json:"id"`
	User    string `json:"user"`
	Movie   string `json:"movie"`
	Theater string `json:"theater"`
	Ticket  string `json:"ticket"`
	Tanggal string `json:"tanggal"`
	Total   int    `json:"total"`
	ScheduleID uint `json:"schedule_id"`
	MovieID uint `json:"movie_id"`
}

func (Transaction) TableName() string {
	return "transaction"
}

func GetTransactionIndex(db *gorm.DB) (TransactionResponse, error) {
	var transactions []Transaction
	var response TransactionResponse

	// Preload semua relasi yang dibutuhkan
	err := db.Model(&Transaction{}).
		Preload("User").
		Preload("Schedule").
		Preload("Schedule.Movie").
		Preload("Schedule.Theater").
		Preload("Details").
		Find(&transactions).Error
	if err != nil {
		return response, err
	}

	// Initialize response
	response.Message = "success"
	response.Data.TotalTransactions = len(transactions)
	response.Data.TicketsSold = 0
	response.Data.TotalRevenue = 0
	response.Data.Transactions = []TransactionItem{}

	// Proses tiap transaksi
	for _, t := range transactions {
		// Hitung jumlah tiket dan total harga
		ticketCount := len(t.Details)
		total := 0
		for _, d := range t.Details {
			total += d.Price
		}

		// Ambil nama movie dan theater
		movieName := ""
		theaterName := ""
		if t.Schedule.Movie != nil {
			movieName = t.Schedule.Movie.Title
		}
		if t.Schedule.Theater != nil {
			theaterName = t.Schedule.Theater.Name
		}

		// Format tanggal
		date := t.CreatedAt.Format("2006-01-02")

		// Format string tiket
		ticketText := fmt.Sprintf("%d Ticket", ticketCount)
		if ticketCount > 1 {
			ticketText = fmt.Sprintf("%d Tickets", ticketCount)
		}

		// Tambahkan ke list transaksi
		response.Data.Transactions = append(response.Data.Transactions, TransactionItem{
			ID:        t.ID,
			ScheduleID: t.ScheduleID,
			User:      t.User.Fullname,
			MovieID:   t.Schedule.MovieID,
			Movie:     movieName,
			Theater:   theaterName,
			Ticket:    ticketText,
			Tanggal:   date,
			Total:     total,
		})

		// Update total
		response.Data.TicketsSold += ticketCount
		response.Data.TotalRevenue += total
	}

	return response, nil
}

