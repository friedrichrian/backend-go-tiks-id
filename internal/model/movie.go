package model

import (
	"time"
)

type Movie struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Title       string     `gorm:"uniqueIndex" json:"title" binding:"required"`
	Description string     `json:"description" binding:"required"`
	Duration    int        `json:"duration" binding:"required,min=1"`
	ReleaseDate string     `json:"release_date" binding:"required"`
	Poster      string     `json:"poster" binding:"omitempty,url"`
	Genres      []Genre    `gorm:"many2many:genre_movie;" json:"genre"`
	Schedules   []Schedule `json:"schedule"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (Movie) TableName() string {
	return "movie"
}
