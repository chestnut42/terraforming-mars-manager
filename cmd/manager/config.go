package main

import (
	"fmt"
	"net/url"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Listen      string `default:":8080"`
	GameURL     URL    `envconfig:"game_url" default:"http://localhost:8090/"`
	PostgresDSN string `envconfig:"postgres_dsn" default:"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"`
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
	url.URL
}

func (u *URL) UnmarshalText(text []byte) error {
	newUrl, err := url.Parse(string(text))
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	u.URL = *newUrl
	return nil
}
