package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type APN struct {
	BaseURL  URL    `envconfig:"base_url" default:"https://api.sandbox.push.apple.com"`
	TeamId   string `envconfig:"team_id"`
	KeyId    string `envconfig:"key_id"`
	KeyFile  string `envconfig:"key_file"`
	BundleId string `envconfig:"bundle_id"`
}

type Notifications struct {
	ActivityBuffer time.Duration `envconfig:"activity_buffer" default:"1h"`
	ScanInterval   time.Duration `envconfig:"scan_interval" default:"5m"`
	WorkersCount   int           `envconfig:"workers_count" default:"10"`
}

type Games struct {
	ScanInterval time.Duration `envconfig:"scan_interval" default:"10m"`
}

type Config struct {
	Listen        string        `default:":8080"`
	GameURL       URL           `envconfig:"game_url" default:"http://localhost:8090/"`
	PostgresDSN   string        `envconfig:"postgres_dsn" default:"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"`
	AppleKeys     string        `envconfig:"apple_keys" default:"https://appleid.apple.com/auth/keys"`
	APN           APN           `envconfig:"apn"`
	Notifications Notifications `envconfig:"notify"`
	Games         Games         `envconfig:"games"`
}

func NewConfig() (Config, error) {
	c := Config{}
	err := envconfig.Process("mars", &c)
	if err != nil {
		return Config{}, fmt.Errorf("unable to parse config: %w", err)
	}
	return c, nil
}

func MustNewConfig() Config {
	c, err := NewConfig()
	if err != nil {
		panic(err)
	}
	return c
}

type URL struct {
	*url.URL
}

func (u *URL) UnmarshalText(text []byte) error {
	newUrl, err := url.Parse(string(text))
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	u.URL = newUrl
	return nil
}
