package model

type MovieGenre struct {
	MovieID uint `gorm:"primaryKey" json:"movie_id"`
	GenreID uint `gorm:"primaryKey" json:"genre_id"`
}

func (MovieGenre) TableName() string {
	return "genre_movie"
}
