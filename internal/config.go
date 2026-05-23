package main

import (
	"fmt"
	"os"
)

type Config struct {
	ModulePath string
	TokenLabel string
	UserPIN    string
	APIKey     string
	Port       string
}

func loadConfig() (Config, error) {
	cfg := Config{
		ModulePath: envOr("PKCS11_MODULE_PATH", "/usr/lib/softhsm/libsofthsm2.so"),
		TokenLabel: envOr("HSM_TOKEN", "dev-ecdsa"),
		UserPIN:    os.Getenv("HSM_USER_PIN"),
		APIKey:     os.Getenv("API_KEY"),
		Port:       envOr("PORT", "8080"),
	}
	if cfg.UserPIN == "" {
		return cfg, fmt.Errorf("HSM_USER_PIN env required")
	}
	if cfg.APIKey == "" {
		return cfg, fmt.Errorf("API_KEY env required")
	}
	return cfg, nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
