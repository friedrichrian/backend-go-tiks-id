package main

import (
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/handler"
	"backend/internal/middleware"
	"backend/internal/model"
	"backend/internal/seed"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// connect DB (MySQL as requested)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db.Connect(dsn)

	// AutoMigrate models
	db.DB.AutoMigrate(&model.User{}, &model.Genre{}, &model.Movie{}, &model.MovieGenre{}, &model.Theater{}, &model.Schedule{}, &model.Transaction{}, &model.TransactionDetail{}, &model.RefreshToken{})

	// seed sample data
	seed.Seed(db.DB)

	r := gin.Default()

	// serve poster files at /posters/<filename>
	r.Static("/posters", "./public/posters")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// auth
	r.POST("/auth/register", handler.Register)
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/refresh", handler.Refresh)

	auth := r.Group("")
	auth.Use(middleware.AuthRequired())
	{
		auth.GET("/auth/user", handler.Me)
		auth.POST("/auth/logout", handler.Logout)
		auth.GET("/auth/me", handler.Me)

		auth.POST("/tickets/book", handler.BookTicket)
		auth.GET("/tickets/my-bookings", handler.GetMyBookings)

		auth.GET("/movie", handler.MovieIndex)
		auth.GET("/theater", handler.TheaterIndex)
		auth.GET("/schedule", handler.ScheduleIndex)
		auth.GET("/genre", handler.GenreIndex)
		auth.GET("/movie/:id", handler.MovieShow)

		admin := auth.Group("")
		admin.Use(middleware.AdminRequired())
		{
			admin.POST("/genre", handler.GenreCreate)
			admin.PATCH("/genre/:id", handler.GenreEdit)
			admin.DELETE("/genre/:id", handler.GenreDelete)

			admin.POST("/schedule", handler.ScheduleCreate)
			admin.PATCH("/schedule/:id", handler.ScheduleEdit)
			admin.DELETE("/schedule/:id", handler.ScheduleDelete)

			admin.POST("/movie", handler.MovieCreate)
			admin.PATCH("/movie/:id", handler.MovieEdit)
			admin.DELETE("/movie/:id", handler.MovieDelete)

			admin.POST("/theater", handler.TheaterCreate)
			admin.PATCH("/theater/:id", handler.TheaterEdit)
			admin.DELETE("/theater/:id", handler.TheaterDelete)
		}
	}

	// Print daftar route
	for _, r := range r.Routes() {
		log.Println(r.Method, r.Path)
	}

	r.Run(fmt.Sprintf(":%s", cfg.AppPort))
}
