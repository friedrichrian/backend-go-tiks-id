package model

import "time"

type TransactionDetail struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TransactionID uint      `json:"transaction_id"`
	Seat          string    `json:"seat"`
	Price         int       `json:"price"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (TransactionDetail) TableName() string {
	return "transaction_detail"
}
