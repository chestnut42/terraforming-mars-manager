package auth

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type User struct {
	Id string
}

type Service struct {
	jwk jwk.Set
}

func NewService(ctx context.Context, keysUrl string) (*Service, error) {
	c := jwk.NewCache(ctx)
	if err := c.Register(keysUrl); err != nil {
		return nil, fmt.Errorf("could not register keys cache: %w", err)
	}
	if _, err := c.Refresh(ctx, keysUrl); err != nil {
		return nil, fmt.Errorf("could not refresh keys cache: %w", err)
	}

	return &Service{
		jwk: jwk.NewCachedSet(c, keysUrl),
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (*User, error) {
	t, err := jwt.ParseString(token, jwt.WithKeySet(s.jwk))
	if err != nil {
		return nil, fmt.Errorf("could not parse token: %w", err)
	}

	return &User{
		Id: t.Subject(),
	}, nil
}
