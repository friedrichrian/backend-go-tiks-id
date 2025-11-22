package seed

import (
	"backend/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) {
	// create admin user if not exists
	var u model.User
	if err := db.First(&u).Error; err != nil {
		pass, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		admin := model.User{Fullname: "Admin", Email: "admin@example.com", Password: string(pass), IsAdmin: true}
		db.Create(&admin)
	}

	// sample genres
	genres := []model.Genre{{Name: "Action"}, {Name: "Drama"}, {Name: "Comedy"}}
	for _, g := range genres {
		var tmp model.Genre
		if db.Where("name = ?", g.Name).First(&tmp).Error != nil {
			db.Create(&g)
		}
	}

	// sample theater
	theaters := []model.Theater{{Name: "A1", Section: "Regular", Row: 5, Col: 10}}
	for _, t := range theaters {
		var tmp model.Theater
		if db.Where("name = ?", t.Name).First(&tmp).Error != nil {
			db.Create(&t)
		}
	}
}
