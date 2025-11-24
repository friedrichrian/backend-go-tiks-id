package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Fullname  string    `json:"fullname"`
	Email     string    `gorm:"uniqueIndex" json:"email"`
	Password  string    `json:"-"`
	IsAdmin   bool      `gorm:"default:false" json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserResponse struct {
	Message string     `json:"message"`
	Data    []UserItem `json:"data"`
}

type UserItem struct {
	ID        uint     `json:"id"`
	Fullname  string   `json:"fullname"`
	Email     string   `json:"email"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// GetAllUsers retrieves all users with formatted dates
func GetAllUsers(db *gorm.DB) (UserResponse, error) {
	var users []User
	var response UserResponse

	if err := db.Find(&users).Error; err != nil {
		return response, err
	}

	response.Message = "success"
	for _, user := range users {
		response.Data = append(response.Data, UserItem{
			ID:        user.ID,
			Fullname:  user.Fullname,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format("2006-01-02"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02"),
		})
	}

	return response, nil
}
