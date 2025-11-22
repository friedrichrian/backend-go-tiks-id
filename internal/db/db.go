package db

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Connect opens a MySQL connection using DSN like "user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
func Connect(dsn string) {
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	// quick ping by getting generic database object
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("failed to get sql DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("failed to ping DB: %v", err)
	}
	fmt.Println("Connected to MySQL")
}
