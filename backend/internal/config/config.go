package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	DatabaseURL, Origin, Address string
	Production                   bool
}

func Load() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), Origin: os.Getenv("APP_ORIGIN"), Address: os.Getenv("LISTEN_ADDR"), Production: os.Getenv("APP_ENV") == "production"}
	if c.Address == "" {
		c.Address = "127.0.0.1:8080"
	}
	u, e := url.Parse(c.Origin)
	if c.DatabaseURL == "" || e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return c, fmt.Errorf("DATABASE_URL y APP_ORIGIN válidos son obligatorios")
	}
	if c.Production && u.Scheme != "https" {
		return c, fmt.Errorf("producción requiere HTTPS")
	}
	return c, nil
}
