package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Server struct {
		Addr string `json:"addr"`
	} `json:"server"`
	DB struct {
		DSN string `json:"dsn"`
	} `json:"db"`
	Redis struct {
		Addr     string `json:"addr"`
		Password string `json:"password"`
		DB       int    `json:"db"`
	} `json:"redis"`
	Upload struct {
		// files live here; mount it as a volume in containers
		Dir   string `json:"dir"`
		MaxMB int    `json:"max_mb"`
	} `json:"upload"`
	JWT struct {
		Secret            string `json:"secret"`
		AccessTTLSeconds  int    `json:"access_ttl_seconds"`
		RefreshTTLSeconds int    `json:"refresh_ttl_seconds"`
		// true whenever users reach the app over HTTPS, including behind Caddy
		CookieSecure bool `json:"cookie_secure"`
	} `json:"jwt"`
}

// C is loaded once at startup and read-only afterwards.
var C Config

// Load reads config/<name>.json relative to the working directory.
func Load(name string) error {
	raw, err := os.ReadFile(filepath.Join("config", name+".json"))
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&C); err != nil {
		return fmt.Errorf("config/%s.json: %w", name, err)
	}
	return validate()
}

func validate() error {
	var missing []error
	if C.Server.Addr == "" {
		missing = append(missing, errors.New("server.addr"))
	}
	if C.DB.DSN == "" {
		missing = append(missing, errors.New("db.dsn"))
	}
	if C.Upload.Dir == "" || C.Upload.MaxMB <= 0 {
		missing = append(missing, errors.New("upload.dir / upload.max_mb"))
	}
	if len(C.JWT.Secret) < 32 {
		missing = append(missing, errors.New("jwt.secret (at least 32 characters)"))
	}
	if C.JWT.AccessTTLSeconds <= 0 || C.JWT.RefreshTTLSeconds <= 0 {
		missing = append(missing, errors.New("jwt.access_ttl_seconds / jwt.refresh_ttl_seconds"))
	}
	if len(missing) > 0 {
		return fmt.Errorf("config incomplete: %w", errors.Join(missing...))
	}
	return nil
}
