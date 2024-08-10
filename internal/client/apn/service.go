package apn

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	maxTokenAge = 30 * time.Second
)

type Config struct {
	TeamId  string
	KeyId   string
	KeyData []byte
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Service struct {
	teamId string
	key    jwk.Key

	now       func() time.Time
	l         sync.Mutex
	token     string
	createdAt time.Time
}

func NewService(cfg Config) (*Service, error) {
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
		teamId: cfg.TeamId,
		key:    key,
	}, nil
}

func (s *Service) getToken() (string, error) {
	s.l.Lock()
	defer s.l.Unlock()

	now := s.now()
	if s.token != "" && now.Sub(s.createdAt) < maxTokenAge {
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
		return "", fmt.
			Errorf("failed to sign token: %w", err)
	}
	s.token = string(signed)
	return s.token, nil
}
