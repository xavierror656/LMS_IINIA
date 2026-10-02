package main

import (
	"aulaquest/internal/database"
	"aulaquest/migrations"
	"log"
	"os"
)

func main() {
	db, e := database.Open(os.Getenv("DATABASE_URL"))
	if e != nil {
		log.Fatal("database unavailable")
	}
	if e = migrations.Apply(db); e != nil {
		log.Fatal(e)
	}
	log.Print("migrations applied")
}
