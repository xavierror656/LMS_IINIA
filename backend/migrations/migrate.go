package migrations

import (
	"embed"
	"fmt"
	"gorm.io/gorm"
	"sort"
)

//go:embed *.sql
var files embed.FS

func Apply(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SELECT pg_advisory_xact_lock(817235)").Error; e != nil {
			return e
		}
		if e := tx.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())").Error; e != nil {
			return e
		}
		entries, e := files.ReadDir(".")
		if e != nil {
			return e
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, f := range entries {
			if f.IsDir() {
				continue
			}
			var n int64
			if e = tx.Raw("SELECT count(*) FROM schema_migrations WHERE name=?", f.Name()).Scan(&n).Error; e != nil {
				return e
			}
			if n > 0 {
				continue
			}
			b, e := files.ReadFile(f.Name())
			if e != nil {
				return e
			}
			if e = tx.Exec(string(b)).Error; e != nil {
				return fmt.Errorf("migration %s: %w", f.Name(), e)
			}
			if e = tx.Exec("INSERT INTO schema_migrations(name) VALUES (?)", f.Name()).Error; e != nil {
				return e
			}
		}
		return nil
	})
}
