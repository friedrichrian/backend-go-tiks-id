package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/db"
	"backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TheaterIndex(c *gin.Context) {
	var t []model.Theater
	db.DB.Find(&t)
	if len(t) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Theater not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Theater found", "data": t})
}

func TheaterCreate(c *gin.Context) {
	// bind to a generic map so we can accept numeric or string for `section`, and coerce row/col
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// validate name
	nameV, ok := payload["name"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"Name": "required"}})
		return
	}
	name, ok := nameV.(string)
	if !ok || strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"Name": "required"}})
		return
	}

	// uniqueness check
	var exists model.Theater
	if err := db.DB.Where("name = ?", name).First(&exists).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"message": "Theater name already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// parse section (accept string or number)
	section := ""
	if sv, ok := payload["section"]; ok {
		switch t := sv.(type) {
		case string:
			section = t
		case float64:
			// JSON numbers are float64
			section = strconv.Itoa(int(t))
		case int:
			section = strconv.Itoa(t)
		default:
			section = fmt.Sprint(t)
		}
	}

	// parse row and col
	row := 0
	if rv, ok := payload["row"]; ok {
		switch v := rv.(type) {
		case float64:
			row = int(v)
		case int:
			row = v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				row = i
			}
		}
	}
	col := 0
	if cv, ok := payload["col"]; ok {
		switch v := cv.(type) {
		case float64:
			col = int(v)
		case int:
			col = v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				col = i
			}
		}
	}

	t := model.Theater{
		Name:    name,
		Section: section,
		Row:     row,
		Col:     col,
	}
	if err := db.DB.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Theater created successfully", "data": t})
}

func TheaterEdit(c *gin.Context) {
	id := c.Param("id")
	var t model.Theater
	if err := db.DB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Theater not found"})
		return
	}

	// accept numeric or string inputs like in create; bind to map and coerce
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// validate name
	nameV, ok := payload["name"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"Name": "required"}})
		return
	}
	name, ok := nameV.(string)
	if !ok || strings.TrimSpace(name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"errors": map[string]string{"Name": "required"}})
		return
	}

	// uniqueness check excluding current
	var exists model.Theater
	if err := db.DB.Where("name = ? AND id <> ?", name, t.ID).First(&exists).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"message": "Theater name already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// parse section
	section := t.Section
	if sv, ok := payload["section"]; ok {
		switch v := sv.(type) {
		case string:
			section = v
		case float64:
			section = strconv.Itoa(int(v))
		case int:
			section = strconv.Itoa(v)
		default:
			section = fmt.Sprint(v)
		}
	}

	// parse row and col
	row := t.Row
	if rv, ok := payload["row"]; ok {
		switch v := rv.(type) {
		case float64:
			row = int(v)
		case int:
			row = v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				row = i
			}
		}
	}
	col := t.Col
	if cv, ok := payload["col"]; ok {
		switch v := cv.(type) {
		case float64:
			col = int(v)
		case int:
			col = v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				col = i
			}
		}
	}

	t.Name = name
	t.Section = section
	t.Row = row
	t.Col = col
	if err := db.DB.Save(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Theater updated successfully", "data": t})
}

func TheaterDelete(c *gin.Context) {
	id := c.Param("id")
	var t model.Theater
	if err := db.DB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Theater not found"})
		return
	}
	db.DB.Delete(&t)
	c.JSON(http.StatusOK, gin.H{"message": "Theater deleted successfully"})
}
