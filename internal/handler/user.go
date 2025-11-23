package handler

import (
	"net/http"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
)

// UserIndex handles GET /users
func UserIndex(c *gin.Context) {
	// Check if user is admin (this is already handled by AdminRequired middleware)
	// Just need to get users

	users, err := model.GetAllUsers(db.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, users)
}
