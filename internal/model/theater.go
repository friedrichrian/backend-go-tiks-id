package model

import "time"

type Theater struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex" json:"name" binding:"required"`
	Section   string    `json:"section"`
	Row       int       `json:"row"`
	Col       int       `json:"col"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Theater) TableName() string {
	return "theater"
}
