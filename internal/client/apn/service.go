package apn

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Config struct {
	BaseURL     *url.URL
	Topic       string
	TeamId      string
	KeyId       string
	KeyData     []byte
	MaxTokenAge time.Duration
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Service struct {
	baseURL *url.URL
	topic   string
	teamId  string
	key     jwk.Key
	maxAge  time.Duration

	client Client

	now       func() time.Time
	l         sync.Mutex
	token     string
	createdAt time.Time
}

func NewService(cfg Config, client Client) (*Service, error) {
	key, err := jwk.ParseKey(cfg.KeyData, jwk.WithPEM(true))
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}
	if err := key.Validate(); err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}
	if err := key.Set("kid", cfg.KeyId); err != nil {
		return nil, fmt.Errorf("failed to set key: %w", err)
	}

	return &Service{
		baseURL: cfg.BaseURL,
		topic:   cfg.Topic,
		teamId:  cfg.TeamId,
		key:     key,
		maxAge:  cfg.MaxTokenAge,

		client: client,
		now:    time.Now,
	}, nil
}

func (s *Service) getToken() (string, error) {
	s.l.Lock()
	defer s.l.Unlock()

	now := s.now()
	if s.token != "" && now.Sub(s.createdAt) < s.maxAge {
		return s.token, nil
	}

	token, err := jwt.NewBuilder().
		Issuer(s.teamId).
		IssuedAt(now).
		Build()
	if err != nil {
		return "", fmt.Errorf("failed to build token: %w", err)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, s.key))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	s.token = string(signed)
	return s.token, nil
}
