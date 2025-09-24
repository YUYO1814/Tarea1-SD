package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"tarea1-uzm/internal/api"
	"tarea1-uzm/internal/db"
)

func main() {
	sqlDB, err := db.Open("data/uzm.db")
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(sqlDB); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	r := gin.Default()
	api.RegisterRoutes(r, sqlDB)

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
