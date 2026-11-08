package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gotest.tools/v3/assert"
)

type testKey struct {
	private jwk.Key
	public  jwk.Key
}

func newTestKey(t *testing.T, kid string) testKey {
	t.Helper()
	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NilError(t, err)

	private, err := jwk.FromRaw(raw)
	assert.NilError(t, err)
	assert.NilError(t, private.Set(jwk.KeyIDKey, kid))
	assert.NilError(t, private.Set(jwk.AlgorithmKey, jwa.RS256))

	public, err := private.PublicKey()
	assert.NilError(t, err)
	return testKey{private: private, public: public}
}

func serveKeys(t *testing.T, keys ...testKey) string {
	t.Helper()
	set := jwk.NewSet()
	for _, k := range keys {
		assert.NilError(t, set.AddKey(k.public))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		assert.NilError(t, json.NewEncoder(w).Encode(set))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

type tokenParams struct {
	issuer    string
	audiences []string
	subject   string
	issuedAt  time.Time
	expires   time.Time
}

func validParams(issuer string, audience string) tokenParams {
	return tokenParams{
		issuer:    issuer,
		audiences: []string{audience},
		subject:   "1234",
		issuedAt:  time.Now(),
		expires:   time.Now().Add(time.Hour),
	}
}

func buildToken(t *testing.T, p tokenParams) jwt.Token {
	t.Helper()
	b := jwt.NewBuilder().
		Issuer(p.issuer).
		Audience(p.audiences).
		Subject(p.subject).
		IssuedAt(p.issuedAt)
	if !p.expires.IsZero() {
		b = b.Expiration(p.expires)
	}
	tok, err := b.Build()
	assert.NilError(t, err)
	return tok
}

func signToken(t *testing.T, key testKey, p tokenParams) string {
	t.Helper()
	signed, err := jwt.Sign(buildToken(t, p), jwt.WithKey(jwa.RS256, key.private))
	assert.NilError(t, err)
	return string(signed)
}

func modify(p tokenParams, f func(p *tokenParams)) tokenParams {
	f(&p)
	return p
}

const (
	appleIssuer  = "https://appleid.apple.com"
	googleIssuer = "https://accounts.google.com"
)

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appleKey := newTestKey(t, "apple")
	googleKey := newTestKey(t, "google")
	// forgedKey reuses Google's kid, so it fails on the signature, not on the kid lookup
	forgedKey := newTestKey(t, "google")
	otherKey := newTestKey(t, "other")

	s, err := NewService(ctx,
		Provider{
			Issuers:   []string{appleIssuer},
			KeysURL:   serveKeys(t, appleKey),
			Audiences: []string{"bundle-id"},
		},
		Provider{
			Issuers:      []string{googleIssuer, "accounts.google.com"},
			KeysURL:      serveKeys(t, googleKey),
			Audiences:    []string{"web-client", "other-client"},
			UserIdPrefix: "google:",
		},
	)
	assert.NilError(t, err)

	apple := validParams(appleIssuer, "bundle-id")
	google := validParams(googleIssuer, "web-client")
	tests := []struct {
		name    string
		key     testKey
		params  tokenParams
		wantId  string
		wantErr string
	}{
		{
			name:   "apple",
			key:    appleKey,
			params: apple,
			wantId: "1234",
		},
		{
			name:   "google",
			key:    googleKey,
			params: google,
			wantId: "google:1234",
		},
		{
			name:   "google short issuer",
			key:    googleKey,
			params: modify(google, func(p *tokenParams) { p.issuer = "accounts.google.com" }),
			wantId: "google:1234",
		},
		{
			name:   "one of many audiences",
			key:    googleKey,
			params: modify(google, func(p *tokenParams) { p.audiences = []string{"unknown", "other-client"} }),
			wantId: "google:1234",
		},
		{
			name:   "issued slightly in the future",
			key:    appleKey,
			params: modify(apple, func(p *tokenParams) { p.issuedAt = time.Now().Add(10 * time.Second) }),
			wantId: "1234",
		},
		{
			name:    "wrong audience",
			key:     appleKey,
			params:  modify(apple, func(p *tokenParams) { p.audiences = []string{"web-client"} }),
			wantErr: "invalid audience",
		},
		{
			name:    "unknown issuer",
			key:     otherKey,
			params:  modify(apple, func(p *tokenParams) { p.issuer = "https://example.com" }),
			wantErr: "unknown issuer",
		},
		{
			name:    "issuer case differs",
			key:     googleKey,
			params:  modify(google, func(p *tokenParams) { p.issuer = "https://Accounts.Google.com" }),
			wantErr: "unknown issuer",
		},
		{
			name:    "expired",
			key:     appleKey,
			params:  modify(apple, func(p *tokenParams) { p.expires = time.Now().Add(-time.Hour) }),
			wantErr: `"exp" not satisfied`,
		},
		{
			name:    "no expiration",
			key:     appleKey,
			params:  modify(apple, func(p *tokenParams) { p.expires = time.Time{} }),
			wantErr: `"exp" not satisfied`,
		},
		{
			name:    "forged signature",
			key:     forgedKey,
			params:  google,
			wantErr: "could not verify",
		},
		{
			name:    "signed by other provider",
			key:     appleKey,
			params:  google,
			wantErr: "could not parse token",
		},
		{
			name:    "no subject",
			key:     googleKey,
			params:  modify(google, func(p *tokenParams) { p.subject = "" }),
			wantErr: "token has no subject",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := s.Authenticate(ctx, signToken(t, tt.key, tt.params))
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, user.Id, tt.wantId)
		})
	}

	t.Run("alg none", func(t *testing.T) {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","kid":"apple"}`))
		payload, err := json.Marshal(buildToken(t, apple))
		assert.NilError(t, err)
		token := header + "." + base64.RawURLEncoding.EncodeToString(payload) + "."

		_, err = s.Authenticate(ctx, token)
		assert.ErrorContains(t, err, "could not parse token")
	})

	t.Run("symmetric key", func(t *testing.T) {
		hmacKey, err := jwk.FromRaw([]byte("secret"))
		assert.NilError(t, err)
		assert.NilError(t, hmacKey.Set(jwk.KeyIDKey, "apple"))
		signed, err := jwt.Sign(buildToken(t, apple), jwt.WithKey(jwa.HS256, hmacKey))
		assert.NilError(t, err)

		_, err = s.Authenticate(ctx, string(signed))
		assert.ErrorContains(t, err, "could not parse token")
	})

	t.Run("no kid", func(t *testing.T) {
		raw := &rsa.PrivateKey{}
		assert.NilError(t, appleKey.private.Raw(raw))
		signed, err := jwt.Sign(buildToken(t, apple), jwt.WithKey(jwa.RS256, raw))
		assert.NilError(t, err)

		_, err = s.Authenticate(ctx, string(signed))
		assert.ErrorContains(t, err, "could not parse token")
	})
}

func TestAuthenticateGarbage(t *testing.T) {
	s, err := NewService(context.Background())
	assert.NilError(t, err)

	_, err = s.Authenticate(context.Background(), "not a token")
	assert.ErrorContains(t, err, "could not parse token")
}

func TestNewServiceValidation(t *testing.T) {
	_, err := NewService(context.Background(), Provider{KeysURL: "http://localhost", Audiences: []string{"a"}})
	assert.ErrorContains(t, err, "no issuers")

	_, err = NewService(context.Background(), Provider{Issuers: []string{"iss"}, KeysURL: "http://localhost", Audiences: []string{""}})
	assert.ErrorContains(t, err, "no valid audiences")

	_, err = NewService(context.Background(), Provider{Issuers: []string{"iss"}, KeysURL: "http://localhost"})
	assert.ErrorContains(t, err, "no valid audiences")

	keysURL := serveKeys(t, newTestKey(t, "kid"))
	_, err = NewService(context.Background(),
		Provider{Issuers: []string{"iss"}, KeysURL: keysURL, Audiences: []string{"a"}},
		Provider{Issuers: []string{"iss"}, KeysURL: keysURL, Audiences: []string{"b"}},
	)
	assert.ErrorContains(t, err, "duplicate issuer")
}
