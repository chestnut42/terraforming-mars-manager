package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type User struct {
	Id string
}

// Provider describes an identity provider whose ID tokens are accepted.
type Provider struct {
	// Issuers are the accepted values of the "iss" claim.
	Issuers []string
	// KeysURL is the JWKS endpoint with the provider signing keys.
	KeysURL string
	// Audiences are the accepted values of the "aud" claim (client IDs).
	Audiences []string
	// UserIdPrefix is prepended to the token subject to form the user ID.
	UserIdPrefix string
}

type provider struct {
	keys         jwk.Set
	audiences    []string
	userIdPrefix string
}

type Service struct {
	byIssuer map[string]*provider
}

func NewService(ctx context.Context, providers ...Provider) (*Service, error) {
	c := jwk.NewCache(ctx)
	byIssuer := make(map[string]*provider)
	for _, p := range providers {
		if len(p.Issuers) == 0 {
			return nil, errors.New("provider has no issuers")
		}
		if len(p.Audiences) == 0 || slices.Contains(p.Audiences, "") {
			return nil, fmt.Errorf("provider %s has no valid audiences", p.Issuers[0])
		}
		if err := c.Register(p.KeysURL); err != nil {
			return nil, fmt.Errorf("could not register keys cache: %w", err)
		}
		if _, err := c.Refresh(ctx, p.KeysURL); err != nil {
			return nil, fmt.Errorf("could not refresh keys cache: %w", err)
		}

		prov := &provider{
			keys:         jwk.NewCachedSet(c, p.KeysURL),
			audiences:    p.Audiences,
			userIdPrefix: p.UserIdPrefix,
		}
		for _, iss := range p.Issuers {
			if _, ok := byIssuer[iss]; ok {
				return nil, fmt.Errorf("duplicate issuer: %s", iss)
			}
			byIssuer[iss] = prov
		}
	}

	return &Service{
		byIssuer: byIssuer,
	}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (*User, error) {
	// Peek at the issuer to pick the provider, the token is verified below
	unverified, err := jwt.ParseInsecure([]byte(token))
	if err != nil {
		return nil, fmt.Errorf("could not parse token: %w", err)
	}
	iss := unverified.Issuer()
	p, ok := s.byIssuer[iss]
	if !ok {
		return nil, fmt.Errorf("unknown issuer: %q", iss)
	}

	t, err := jwt.ParseString(token, jwt.WithKeySet(p.keys), jwt.WithIssuer(iss))
	if err != nil {
		return nil, fmt.Errorf("could not parse token: %w", err)
	}
	if !slices.ContainsFunc(t.Audience(), func(aud string) bool {
		return slices.Contains(p.audiences, aud)
	}) {
		return nil, fmt.Errorf("invalid audience: %v", t.Audience())
	}
	if t.Subject() == "" {
		return nil, errors.New("token has no subject")
	}

	return &User{
		Id: p.userIdPrefix + t.Subject(),
	}, nil
}
