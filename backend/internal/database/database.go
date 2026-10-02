package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

func Open(dsn string) (*gorm.DB, error) {
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		return nil, e
	}
	pool, e := db.DB()
	if e != nil {
		return nil, e
	}
	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(3)
	pool.SetConnMaxLifetime(30 * time.Minute)
	return db, pool.Ping()
}
