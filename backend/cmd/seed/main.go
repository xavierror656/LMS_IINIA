package main

import (
	"aulaquest/internal/database"
	"aulaquest/internal/services"
	"log"
	"os"
)

func main() {
	db, e := database.Open(os.Getenv("DATABASE_URL"))
	if e != nil {
		log.Fatal("database unavailable")
	}
	if e = services.Seed(db); e != nil {
		log.Fatal(e)
	}
	log.Print("synthetic seed applied; existing accounts unchanged")
}
