package model

import "time"

type Transaction struct {
	ID              uint                `gorm:"primaryKey" json:"id"`
	UserID          uint                `json:"user_id"`
	ScheduleID      uint                `json:"schedule_id"`
	TransactionTime time.Time           `json:"transaction_time"`
	Details         []TransactionDetail `json:"transaction_details"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
}

func (Transaction) TableName() string {
	return "transaction"
}
