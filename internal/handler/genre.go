package handler

import (
	"errors"
	"net/http"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

func GenreIndex(c *gin.Context) {
	var genres []model.Genre
	db.DB.Find(&genres)
	if len(genres) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Genre not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Genre found", "data": genres})
}

func GenreCreate(c *gin.Context) {
	var payload struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make(map[string]string)
			for _, fe := range ve {
				out[fe.Field()] = fe.Tag()
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": out})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// check uniqueness
	var exists model.Genre
	if err := db.DB.Where("name = ?", payload.Name).First(&exists).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"message": "Genre already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	g := model.Genre{Name: payload.Name}
	if err := db.DB.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Genre created successfully", "data": g})
}

func GenreEdit(c *gin.Context) {
	id := c.Param("id")
	var g model.Genre
	if err := db.DB.First(&g, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Genre not found"})
		return
	}
	var payload struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			out := make(map[string]string)
			for _, fe := range ve {
				out[fe.Field()] = fe.Tag()
			}
			c.JSON(http.StatusBadRequest, gin.H{"errors": out})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// check uniqueness excluding current
	var exists model.Genre
	if err := db.DB.Where("name = ? AND id <> ?", payload.Name, g.ID).First(&exists).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"message": "Genre already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	g.Name = payload.Name
	if err := db.DB.Save(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Genre updated successfully", "data": g})
}

func GenreDelete(c *gin.Context) {
	id := c.Param("id")
	var g model.Genre
	if err := db.DB.First(&g, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Genre not found"})
		return
	}
	db.DB.Delete(&g)
	c.JSON(http.StatusOK, gin.H{"message": "Genre deleted successfully"})
}
