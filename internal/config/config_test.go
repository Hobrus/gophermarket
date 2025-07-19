package config

import (
	"os"
	"testing"
)

func TestLoad_FlagsOverrideEnv(t *testing.T) {
	t.Setenv("RUN_ADDRESS", ":1111")
	t.Setenv("DATABASE_URI", "envdb")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "envacc")
	t.Setenv("JWT_SECRET", "envjwt")

	os.Args = []string{"cmd", "-a", ":2222", "-d", "flagdb", "-r", "flagacc", "-s", "flagjwt"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.RunAddress != ":2222" {
		t.Errorf("expected run address from flag, got %s", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "flagdb" {
		t.Errorf("expected database uri from flag, got %s", cfg.DatabaseURI)
	}
	if cfg.AccrualAddress != "flagacc" {
		t.Errorf("expected accrual address from flag, got %s", cfg.AccrualAddress)
	}
	if cfg.JWTSecret != "flagjwt" {
		t.Errorf("expected jwt secret from flag, got %s", cfg.JWTSecret)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	t.Setenv("RUN_ADDRESS", "")
	t.Setenv("DATABASE_URI", "")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "")
	t.Setenv("JWT_SECRET", "")

	os.Args = []string{"cmd"}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
