package config

import "testing"

func TestProductionRequiresHTTPS(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://local/test")
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_ORIGIN", "http://localhost:4321")
	if _, e := Load(); e == nil {
		t.Fatal("insecure production accepted")
	}
	t.Setenv("APP_ORIGIN", "https://school.example")
	if _, e := Load(); e != nil {
		t.Fatal(e)
	}
}
func TestOriginRejectsPath(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://local/test")
	t.Setenv("APP_ORIGIN", "https://school.example/path")
	if _, e := Load(); e == nil {
		t.Fatal("origin with path accepted")
	}
}
