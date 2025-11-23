package model

import (
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
	User    string `json:"user"`
	Movie   string `json:"movie"`
	Theater string `json:"theater"`
	Ticket  string `json:"ticket"`
	Tanggal string `json:"tanggal"`
	Total   int    `json:"total"`
}

func (Transaction) TableName() string {
	return "transaction"
}

func GetTransactionIndex(db *gorm.DB) (TransactionResponse, error) {
	var transactions []Transaction
	var response TransactionResponse

	// Preload all necessary relationships
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

	// Process each transaction
	for _, t := range transactions {
		// Calculate tickets and total for this transaction
		tickets := len(t.Details)
		total := 0
		for _, d := range t.Details {
			total += d.Price
		}

		// Get movie and theater names
		movieName := ""
		theaterName := ""
		if t.Schedule.Movie != nil {
			movieName = t.Schedule.Movie.Title
		}
		if t.Schedule.Theater != nil {
			theaterName = t.Schedule.Theater.Name
		}

		// Format date
		date := t.CreatedAt.Format("2006-01-02")

		// Add to transactions list
		response.Data.Transactions = append(response.Data.Transactions, TransactionItem{
			User:    t.User.Fullname,
			Movie:   movieName,
			Theater: theaterName,
			Ticket:  "1 Ticket", // Default, will be updated in next step
			Tanggal: date,
			Total:   total,
		})

		// Update totals
		response.Data.TicketsSold += tickets
		response.Data.TotalRevenue += total
	}

	// Update ticket counts in the response
	for i, t := range transactions {
		tickets := len(t.Details)
		response.Data.Transactions[i].Ticket = "1 Ticket"
		if tickets > 1 {
			response.Data.Transactions[i].Ticket = "2 Tickets"
		}
	}

	return response, nil
}
