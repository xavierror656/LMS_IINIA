package main

import (
	"aulaquest/internal/database"
	"aulaquest/internal/repositories"
	"aulaquest/internal/services"
	"encoding/json"
	"flag"
	"log"
	"os"
)

// retention applies the documented policy to abandoned draft files. It reports
// before deleting and deletes only with an explicit -confirm flag, so a cron job
// can never destroy work by accident.
func main() {
	days := flag.Int("days", 180, "age in days after which an untouched draft counts as abandoned")
	confirm := flag.Bool("confirm", false, "delete the reported files; without it the command only reports")
	flag.Parse()
	if *days < 1 {
		log.Fatal("days must be positive")
	}
	dsn := os.Getenv("DATABASE_URL")
	db, e := database.Open(dsn)
	if e != nil {
		log.Fatal("database unavailable")
	}
	academic := services.AcademicService{Repo: repositories.Repository{DB: db}}
	var report services.RetentionReport
	if *confirm {
		report, e = academic.PurgeAbandonedDrafts(*days)
	} else {
		report, e = academic.RetentionPreview(*days)
	}
	if e != nil {
		log.Fatal(e)
	}
	out, e := json.Marshal(report)
	if e != nil {
		log.Fatal(e)
	}
	os.Stdout.Write(append(out, '\n'))
	if !*confirm {
		log.Print("informe sin cambios: usa -confirm para borrar los archivos enumerados")
	}
}
