package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"backend/internal/db"
	"backend/internal/model"
	dto "backend/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// request DTOs to avoid binding tags on model.Movie being applied to multipart form
type movieForm struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description" binding:"required"`
	Duration    int    `form:"duration" binding:"required,min=1"`
	ReleaseDate string `form:"release_date" binding:"required"`
	GenreStr    string `form:"genre" binding:"-"`
}

type movieJSON struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Duration    int     `json:"duration" binding:"required,min=1"`
	ReleaseDate string  `json:"release_date" binding:"required"`
	GenreIDs    []uint  `json:"genre_ids" binding:"-"`
	Poster      *string `json:"poster" binding:"omitempty,url"`
}

func MovieIndex(c *gin.Context) {
	pageStr := c.Query("page")
	perPageStr := c.Query("per_page")
	page := 1
	perPage := 10
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
		perPage = pp
	}

	var movies []model.Movie
	offset := (page - 1) * perPage
	var total int64
	db.DB.Model(&model.Movie{}).Count(&total)

	// cap perPage to avoid excessive load
	if perPage > 100 {
		perPage = 100
	}

	result := db.DB.Preload("Genres").Limit(perPage).Offset(offset).Find(&movies)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	var responses []dto.MovieResponse

	for _, m := range movies {
		var genreNames []string
		for _, g := range m.Genres {
			genreNames = append(genreNames, g.Name)
		}

		// build poster URL
		posterURL := ""
		if m.Poster != "" {
			scheme := "http"
			if c.Request.TLS != nil {
				scheme = "https"
			}
			posterURL = fmt.Sprintf("%s://%s/posters/%s", scheme, c.Request.Host, filepath.Base(m.Poster))
		}

		res := dto.MovieResponse{
			ID:          m.ID,
			Title:       m.Title,
			Description: m.Description,
			Duration:    m.Duration,
			ReleaseDate: m.ReleaseDate,
			Poster:      posterURL,
			Genres:      genreNames,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		}

		responses = append(responses, res)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Movies found",
		"data":    responses,
		"meta": gin.H{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func MovieCreate(c *gin.Context) {
	// support both JSON and multipart/form-data (for file upload)
	contentType := c.GetHeader("Content-Type")
	var m model.Movie
	var genresIDs []uint
	if strings.HasPrefix(contentType, "multipart/form-data") {
		var form movieForm
		if err := c.ShouldBind(&form); err != nil {
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

		// parse genres from comma-separated string
		if form.GenreStr != "" {
			parts := strings.Split(form.GenreStr, ",")
			for _, p := range parts {
				if p == "" {
					continue
				}
				if id, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
					genresIDs = append(genresIDs, uint(id))
				}
			}
		}

		// handle file upload (optional)
		posterPath := ""
		if file, err := c.FormFile("poster"); err == nil {
			uploadDir := filepath.Join("public", "posters")
			if err := os.MkdirAll(uploadDir, 0755); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload dir"})
				return
			}
			uniqueName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
			dst := filepath.Join(uploadDir, uniqueName)
			if err := c.SaveUploadedFile(file, dst); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save poster"})
				return
			}
			posterPath = dst
		}

		m = model.Movie{
			Title:       form.Title,
			Description: form.Description,
			Duration:    form.Duration,
			ReleaseDate: form.ReleaseDate,
			Poster:      posterPath,
		}

	} else {
		var in movieJSON
		if err := c.ShouldBindJSON(&in); err != nil {
			var ve validator.ValidationErrors
			if errors.As(err, &ve) {
				out := make(map[string]string)
				for _, fe := range ve {
					out[fe.Field()] = fe.Tag()
				}
				c.JSON(http.StatusBadRequest, gin.H{"errors": out})
				return
			}
			msg := err.Error()
			if strings.Contains(msg, "invalid character '-' in numeric literal") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: looks like a date or string was not quoted (e.g. use \"release_date\": \"2008-07-18 00:00:00\")"})
				return
			}
			if strings.Contains(msg, "cannot unmarshal") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON type in request: " + msg})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}

		m = model.Movie{
			Title:       in.Title,
			Description: in.Description,
			Duration:    in.Duration,
			ReleaseDate: in.ReleaseDate,
		}
		if in.Poster != nil {
			m.Poster = *in.Poster
		}
		if len(in.GenreIDs) > 0 {
			genresIDs = in.GenreIDs
		}
	}

	// validate unique title before create
	var existing model.Movie
	if err := db.DB.Where("title = ?", m.Title).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"message": "Movie title already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// create movie
	if err := db.DB.Create(&m).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// attach genres if provided (form) or if JSON provided with Genre IDs
	if len(genresIDs) > 0 {
		var genres []model.Genre
		if err := db.DB.Where("id IN ?", genresIDs).Find(&genres).Error; err == nil {
			db.DB.Model(&m).Association("Genres").Replace(genres)
			// reload
			db.DB.Preload("Genres").First(&m, m.ID)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Movie created successfully", "data": m})
}

func MovieShow(c *gin.Context) {
	id := c.Param("id")
	var m model.Movie
	if err := db.DB.Preload("Genres").Preload("Schedules.Theater").First(&m, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Movie not found"})
		return
	}

	// Build grouped response: theaters -> dates -> times (with filled seats)
	type timeEntry struct {
		ScheduleID  uint     `json:"schedule_id"`
		Time        string   `json:"time"`
		FilledSeats []string `json:"filled_seats"`
	}
	type dateEntry struct {
		Date           string      `json:"date"`
		AvailableTimes []timeEntry `json:"available_times"`
	}
	type theaterEntry struct {
		TheaterID      uint        `json:"theater_id"`
		Theater        string      `json:"theater"`
		Section        int         `json:"section"`
		Row            int         `json:"row"`
		Column         int         `json:"column"`
		AvailableDates []dateEntry `json:"available_dates"`
	}

	theatersMap := map[uint]*theaterEntry{}
	var theaterOrder []uint

	for _, s := range m.Schedules {
		// parse start time
		t, err := time.Parse("2006-01-02 15:04:05", s.StartTime)
		dateStr := s.StartTime
		timeStr := s.StartTime
		if err == nil {
			dateStr = t.Format("02 Jan 2006")
			timeStr = t.Format("15:04")
		}

		// collect filled seats for this schedule
		var transactions []model.Transaction
		filledSeats := []string{} // Initialize as empty slice instead of nil
		if err := db.DB.Preload("Details").Where("schedule_id = ?", s.ID).Find(&transactions).Error; err == nil {
			filledSet := map[string]struct{}{}
			for _, tr := range transactions {
				for _, d := range tr.Details {
					if d.Seat != "" {
						filledSet[d.Seat] = struct{}{}
					}
				}
			}
			// Convert set to slice
			for seat := range filledSet {
				filledSeats = append(filledSeats, seat)
			}
		}

		// ensure theater entry exists
		te, ok := theatersMap[s.TheaterID]
		if !ok {
			sec := 0
			if secVal, err := strconv.Atoi(s.Theater.Section); err == nil {
				sec = secVal
			}
			te = &theaterEntry{
				TheaterID: s.TheaterID,
				Theater:   s.Theater.Name,
				Section:   sec,
				Row:       s.Theater.Row,
				Column:    s.Theater.Col,
			}
			theatersMap[s.TheaterID] = te
			theaterOrder = append(theaterOrder, s.TheaterID)
		}

		// find or create date entry in te
		var de *dateEntry
		for i := range te.AvailableDates {
			if te.AvailableDates[i].Date == dateStr {
				de = &te.AvailableDates[i]
				break
			}
		}
		if de == nil {
			te.AvailableDates = append(te.AvailableDates, dateEntry{Date: dateStr})
			de = &te.AvailableDates[len(te.AvailableDates)-1]
		}

		// append time entry
		de.AvailableTimes = append(de.AvailableTimes, timeEntry{
			ScheduleID:  s.ID,
			Time:        timeStr,
			FilledSeats: filledSeats,
		})
	}

	// sort theaterOrder for deterministic output
	sort.SliceStable(theaterOrder, func(i, j int) bool { return theaterOrder[i] < theaterOrder[j] })

	var availableTheaters []theaterEntry
	for _, tid := range theaterOrder {
		// also sort dates within theater by date string
		te := theatersMap[tid]
		sort.SliceStable(te.AvailableDates, func(i, j int) bool { return te.AvailableDates[i].Date < te.AvailableDates[j].Date })
		// sort times inside each date by time
		for di := range te.AvailableDates {
			sort.SliceStable(te.AvailableDates[di].AvailableTimes, func(a, b int) bool {
				return te.AvailableDates[di].AvailableTimes[a].Time < te.AvailableDates[di].AvailableTimes[b].Time
			})
		}
		availableTheaters = append(availableTheaters, *te)
	}

	// build poster url
	posterURL := ""
	if m.Poster != "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		posterURL = fmt.Sprintf("%s://%s/posters/%s", scheme, c.Request.Host, filepath.Base(m.Poster))
	}
	// genres as array of names
	var genreNames []string
	for _, g := range m.Genres {
		genreNames = append(genreNames, g.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Movie found",
		"data": gin.H{
			"title":              m.Title,
			"description":        m.Description,
			"genres":             genreNames,
			"duration":           m.Duration,
			"release_date":       m.ReleaseDate,
			"poster":             posterURL,
			"available_theaters": availableTheaters,
		},
	})
}

func MovieEdit(c *gin.Context) {
	id := c.Param("id")
	var m model.Movie
	if err := db.DB.Preload("Genres").First(&m, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Movie not found"})
		return
	}
	// Support multipart/form-data PATCH (optional fields) and JSON PATCH
	contentType := c.GetHeader("Content-Type")
	// multipart/form-data branch
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// parse select form values and update only provided fields
		if title := c.PostForm("title"); title != "" {
			var exists model.Movie
			if err := db.DB.Where("title = ? AND id <> ?", title, m.ID).First(&exists).Error; err == nil {
				c.JSON(http.StatusConflict, gin.H{"message": "Movie title already exists"})
				return
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			m.Title = title
		}
		if desc := c.PostForm("description"); desc != "" {
			m.Description = desc
		}
		if dur := c.PostForm("duration"); dur != "" {
			if di, err := strconv.Atoi(dur); err == nil {
				m.Duration = di
			}
		}
		if rd := c.PostForm("release_date"); rd != "" {
			m.ReleaseDate = rd
		}

		// genres: comma-separated IDs
		if gstr := c.PostForm("genre"); gstr != "" {
			var ids []uint
			parts := strings.Split(gstr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if idn, err := strconv.Atoi(p); err == nil {
					ids = append(ids, uint(idn))
				}
			}
			if len(ids) > 0 {
				var genres []model.Genre
				if err := db.DB.Where("id IN ?", ids).Find(&genres).Error; err == nil {
					db.DB.Model(&m).Association("Genres").Replace(genres)
				}
			}
		}

		// poster file optionally
		if file, err := c.FormFile("poster"); err == nil {
			uploadDir := filepath.Join("public", "posters")
			if err := os.MkdirAll(uploadDir, 0755); err == nil {
				uniqueName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file.Filename))
				dst := filepath.Join(uploadDir, uniqueName)
				if err := c.SaveUploadedFile(file, dst); err == nil {
					// remove old poster if exists
					if m.Poster != "" {
						_ = os.Remove(m.Poster)
						_ = os.Remove(filepath.Join("public", "posters", filepath.Base(m.Poster)))
					}
					m.Poster = dst
				}
			}
		}

		if err := db.DB.Save(&m).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		db.DB.Preload("Genres").First(&m, id)
		c.JSON(http.StatusOK, gin.H{"message": "Movie updated successfully", "data": m})
		return
	}

	// JSON branch: allow partial updates via map
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if v, ok := payload["title"].(string); ok {
		// ensure title uniqueness (exclude current movie)
		var exists model.Movie
		if err := db.DB.Where("title = ? AND id <> ?", v, m.ID).First(&exists).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"message": "Movie title already exists"})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		updates["title"] = v
	}
	if v, ok := payload["description"].(string); ok {
		updates["description"] = v
	}
	if v, ok := payload["duration"]; ok {
		switch t := v.(type) {
		case float64:
			updates["duration"] = int(t)
		case int:
			updates["duration"] = t
		}
	}
	if v, ok := payload["release_date"].(string); ok {
		updates["release_date"] = v
	}
	if v, ok := payload["poster"].(string); ok {
		updates["poster"] = v
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&m).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// handle genre_ids array in JSON
	if gids, ok := payload["genre_ids"].([]interface{}); ok {
		var ids []uint
		for _, gi := range gids {
			if nid, ok := gi.(float64); ok {
				ids = append(ids, uint(nid))
			}
		}
		if len(ids) > 0 {
			var genres []model.Genre
			if err := db.DB.Where("id IN ?", ids).Find(&genres).Error; err == nil {
				db.DB.Model(&m).Association("Genres").Replace(genres)
			}
		}
	}

	db.DB.Preload("Genres").First(&m, id)
	c.JSON(http.StatusOK, gin.H{"message": "Movie updated successfully", "data": m})
}

func MovieDelete(c *gin.Context) {
	id := c.Param("id")
	var m model.Movie
	if err := db.DB.First(&m, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Movie not found"})
		return
	}
	// attempt to remove poster file if present (ignore not-exist errors)
	if m.Poster != "" {
		if err := os.Remove(m.Poster); err != nil {
			if !os.IsNotExist(err) {
				// try fallback location `public/posters/<basename>`
				alt := filepath.Join("public", "posters", filepath.Base(m.Poster))
				if err2 := os.Remove(alt); err2 != nil && !os.IsNotExist(err2) {
					// ignore failures to remove file — proceed with DB delete
				}
			}
		}
	}

	db.DB.Delete(&m)
	c.JSON(http.StatusOK, gin.H{"message": "Movie deleted successfully"})
}
