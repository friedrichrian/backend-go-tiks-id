package handler

import (
	"net/http"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
)

// GetTransactionIndex handles GET /transactions
func GetTransactionIndex(c *gin.Context) {
	transactions, err := model.GetTransactionIndex(db.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	c.JSON(http.StatusOK, transactions)
}
