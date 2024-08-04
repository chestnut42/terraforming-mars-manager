package mars

import (
	"net/http"
	"net/url"
)

type Service struct {
	baseURL *url.URL
	client  *http.Client
}

func NewService(baseURL *url.URL, client *http.Client) (*Service, error) {
	return &Service{
		baseURL: baseURL,
		client:  client,
	}, nil
}
