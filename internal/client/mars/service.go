package mars

import (
	"net/http"
	"net/url"
)

type Config struct {
	BaseURL       *url.URL
	PublicBaseURL *url.URL
}

type Service struct {
	cfg    Config
	client *http.Client
}

func NewService(cfg Config, client *http.Client) (*Service, error) {
	return &Service{
		cfg:    cfg,
		client: client,
	}, nil
}
