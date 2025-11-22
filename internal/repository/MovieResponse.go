package dto

import (
	"time"
)

type MovieResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Duration    int       `json:"duration"`
	ReleaseDate string    `json:"release_date"`
	Poster      string    `json:"poster"`
	Genres      []string  `json:"genre"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
