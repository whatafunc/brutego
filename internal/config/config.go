// Package config provides configuration loading from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the service.
type Config struct {
	// GRPCAddr is the address the gRPC server listens on.
	GRPCAddr string // default: :50051

	// HTTPAddr is the address the grpc-gateway HTTP server listens on.
	HTTPAddr string // default: :8080

	// LoginRPM is the max allowed authorization attempts per minute per login (N).
	LoginRPM int // default: 10

	// PasswordRPM is the max allowed attempts per minute per password (M).
	PasswordRPM int // default: 100

	// IPRPM is the max allowed attempts per minute per IP (K).
	IPRPM int // default: 1000
}

// New loads configuration from environment variables with sensible defaults.
//
// Environment variables:
//
//	GRPC_ADDR       — gRPC listen address     (default: :50051)
//	HTTP_ADDR       — HTTP gateway address     (default: :8080)
//	LIMIT_LOGIN     — max attempts/min/login   (default: 10)
//	LIMIT_PASSWORD  — max attempts/min/passwd  (default: 100)
//	LIMIT_IP        — max attempts/min/ip      (default: 1000)
func New() (*Config, error) {
	cfg := &Config{
		GRPCAddr:    envString("GRPC_ADDR", ":50051"),
		HTTPAddr:    envString("HTTP_ADDR", ":8080"),
		LoginRPM:    1,
		PasswordRPM: 100,
		IPRPM:       1000,
	}

	var err error

	if cfg.LoginRPM, err = envInt("LIMIT_LOGIN", cfg.LoginRPM); err != nil {
		return nil, fmt.Errorf("LIMIT_LOGIN: %w", err)
	}
	if cfg.PasswordRPM, err = envInt("LIMIT_PASSWORD", cfg.PasswordRPM); err != nil {
		return nil, fmt.Errorf("LIMIT_PASSWORD: %w", err)
	}
	if cfg.IPRPM, err = envInt("LIMIT_IP", cfg.IPRPM); err != nil {
		return nil, fmt.Errorf("LIMIT_IP: %w", err)
	}

	if cfg.LoginRPM <= 0 || cfg.PasswordRPM <= 0 || cfg.IPRPM <= 0 {
		return nil, fmt.Errorf("rate limits must be positive (got login=%d password=%d ip=%d)",
			cfg.LoginRPM, cfg.PasswordRPM, cfg.IPRPM)
	}

	return cfg, nil
}

func envString(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envInt(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value %q: %w", v, err)
	}
	return n, nil
}
