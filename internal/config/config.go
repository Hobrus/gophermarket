package config

import (
	"errors"
	"flag"
	"os"
)

// Config holds application configuration parameters.
type Config struct {
	RunAddress     string
	DatabaseURI    string
	AccrualAddress string
	JWTSecret      string
}

// Load reads configuration from environment variables and command line flags.
// Flags have priority over environment variables.
func Load() (Config, error) {
	cfg := Config{RunAddress: ":8080"}

	if v, ok := os.LookupEnv("RUN_ADDRESS"); ok {
		cfg.RunAddress = v
	}
	if v, ok := os.LookupEnv("DATABASE_URI"); ok {
		cfg.DatabaseURI = v
	}
	if v, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		cfg.AccrualAddress = v
	}
	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
		cfg.JWTSecret = v
	}

	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "run address")
	fs.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "database uri")
	fs.StringVar(&cfg.AccrualAddress, "r", cfg.AccrualAddress, "accrual system address")
	fs.StringVar(&cfg.JWTSecret, "s", cfg.JWTSecret, "jwt secret")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	if cfg.DatabaseURI == "" {
		return Config{}, errors.New("database URI is required")
	}
	if cfg.AccrualAddress == "" {
		return Config{}, errors.New("accrual address is required")
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "secret"
	}

	return cfg, nil
}
