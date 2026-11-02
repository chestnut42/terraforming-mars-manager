package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
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
	issuer   string
	audience string
	subject  string
	expires  time.Time
}

func signToken(t *testing.T, key testKey, p tokenParams) string {
	t.Helper()
	tok, err := jwt.NewBuilder().
		Issuer(p.issuer).
		Audience([]string{p.audience}).
		Subject(p.subject).
		IssuedAt(time.Now()).
		Expiration(p.expires).
		Build()
	assert.NilError(t, err)

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, key.private))
	assert.NilError(t, err)
	return string(signed)
}

func TestAuthenticate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appleKey := newTestKey(t, "apple")
	googleKey := newTestKey(t, "google")
	otherKey := newTestKey(t, "other")

	s, err := NewService(ctx,
		Provider{
			Issuers:   []string{"https://appleid.apple.com"},
			KeysURL:   serveKeys(t, appleKey),
			Audiences: []string{"bundle-id"},
		},
		Provider{
			Issuers:      []string{"https://accounts.google.com", "accounts.google.com"},
			KeysURL:      serveKeys(t, googleKey),
			Audiences:    []string{"web-client", "other-client"},
			UserIdPrefix: "google:",
		},
	)
	assert.NilError(t, err)

	future := time.Now().Add(time.Hour)
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
			params: tokenParams{issuer: "https://appleid.apple.com", audience: "bundle-id", subject: "001.abc", expires: future},
			wantId: "001.abc",
		},
		{
			name:   "google",
			key:    googleKey,
			params: tokenParams{issuer: "https://accounts.google.com", audience: "web-client", subject: "1234", expires: future},
			wantId: "google:1234",
		},
		{
			name:   "google short issuer",
			key:    googleKey,
			params: tokenParams{issuer: "accounts.google.com", audience: "other-client", subject: "1234", expires: future},
			wantId: "google:1234",
		},
		{
			name:    "wrong audience",
			key:     appleKey,
			params:  tokenParams{issuer: "https://appleid.apple.com", audience: "web-client", subject: "001.abc", expires: future},
			wantErr: "invalid audience",
		},
		{
			name:    "unknown issuer",
			key:     otherKey,
			params:  tokenParams{issuer: "https://example.com", audience: "bundle-id", subject: "001.abc", expires: future},
			wantErr: "unknown issuer",
		},
		{
			name:    "expired",
			key:     appleKey,
			params:  tokenParams{issuer: "https://appleid.apple.com", audience: "bundle-id", subject: "001.abc", expires: time.Now().Add(-time.Hour)},
			wantErr: "could not parse token",
		},
		{
			name:    "signed by other provider",
			key:     appleKey,
			params:  tokenParams{issuer: "https://accounts.google.com", audience: "web-client", subject: "1234", expires: future},
			wantErr: "could not parse token",
		},
		{
			name:    "no subject",
			key:     googleKey,
			params:  tokenParams{issuer: "https://accounts.google.com", audience: "web-client", expires: future},
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
}
