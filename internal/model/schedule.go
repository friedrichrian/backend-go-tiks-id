package model

import "time"

type Schedule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MovieID   uint      `json:"movie_id"`
	Movie     *Movie    `json:"movie,omitempty"`
	TheaterID uint      `json:"theater_id"`
	Theater   *Theater  `json:"theater,omitempty"`
	StartTime string    `json:"start_time"`
	Price     int       `json:"price"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Schedule) TableName() string {
	return "schedule"
}
