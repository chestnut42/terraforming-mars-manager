package main

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestNewConfig(t *testing.T) {
	_, err := NewConfig()
	assert.NilError(t, err)
}

func TestConfig(t *testing.T) {
	t.Setenv("MARS_LISTEN", ":42")
	t.Setenv("MARS_GAME_URL", "https://example.com/url")
	t.Setenv("MARS_PUBLIC_GAME_URL", "https://some.other.domain/url")
	t.Setenv("MARS_POSTGRES_DSN", "postgres://somedb")
	t.Setenv("MARS_APN_TEAM_ID", "team id")
	t.Setenv("MARS_APN_KEY_ID", "key id")
	t.Setenv("MARS_APN_KEY_FILE", "key file")
	t.Setenv("MARS_APN_BUNDLE_ID", "bundle-id")
	t.Setenv("MARS_NOTIFY_SCAN_INTERVAL", "42s")
	t.Setenv("MARS_GOOGLE_CLIENT_IDS", "client-1,client-2")

	c, err := NewConfig()
	assert.NilError(t, err)
	assert.Equal(t, c.Listen, ":42")
	assert.Equal(t, c.GameURL.String(), "https://example.com/url")
	assert.Equal(t, c.PublicGameURL.String(), "https://some.other.domain/url")
	assert.Equal(t, c.PostgresDSN, "postgres://somedb")
	assert.Equal(t, c.APN.SandboxURL.String(), "https://api.sandbox.push.apple.com")
	assert.Equal(t, c.APN.ProdURL.String(), "https://api.push.apple.com")
	assert.Equal(t, c.APN.TeamId, "team id")
	assert.Equal(t, c.APN.KeyId, "key id")
	assert.Equal(t, c.APN.KeyFile, "key file")
	assert.Equal(t, c.APN.BundleId, "bundle-id")
	assert.Equal(t, c.APN.MaxTokenAge, 30*time.Minute)
	assert.Equal(t, c.Notifications.ActivityBuffer, time.Hour)
	assert.Equal(t, c.Notifications.ScanInterval, 42*time.Second)
	assert.Equal(t, c.Notifications.WorkersCount, 10)
	assert.Equal(t, c.Games.ScanInterval, 10*time.Minute)
	assert.Equal(t, c.Apple.Keys, "https://appleid.apple.com/auth/keys")
	assert.DeepEqual(t, c.Apple.Audiences, []string(nil))
	assert.Equal(t, c.Google.Keys, "https://www.googleapis.com/oauth2/v3/certs")
	assert.DeepEqual(t, c.Google.ClientIds, []string{"client-1", "client-2"})
}

func TestAuthProviders(t *testing.T) {
	t.Setenv("MARS_APN_BUNDLE_ID", "bundle-id")
	t.Setenv("MARS_GOOGLE_CLIENT_IDS", " client-1, client-2,")

	c, err := NewConfig()
	assert.NilError(t, err)
	providers, err := authProviders(c)
	assert.NilError(t, err)
	assert.Equal(t, len(providers), 2)
	assert.DeepEqual(t, providers[0].Audiences, []string{"bundle-id"})
	assert.DeepEqual(t, providers[1].Audiences, []string{"client-1", "client-2"})
	assert.Equal(t, providers[1].UserIdPrefix, "google:")
}

func TestAuthProvidersAppleAudiences(t *testing.T) {
	t.Setenv("MARS_APN_BUNDLE_ID", "bundle-id")
	t.Setenv("MARS_APPLE_AUDIENCES", "app-1, app-2")

	c, err := NewConfig()
	assert.NilError(t, err)
	providers, err := authProviders(c)
	assert.NilError(t, err)
	assert.Equal(t, len(providers), 1)
	assert.DeepEqual(t, providers[0].Audiences, []string{"app-1", "app-2"})
}

func TestAuthProvidersNoAppleAudiences(t *testing.T) {
	t.Setenv("MARS_APN_BUNDLE_ID", " ")
	t.Setenv("MARS_APPLE_AUDIENCES", ",")

	c, err := NewConfig()
	assert.NilError(t, err)
	_, err = authProviders(c)
	assert.ErrorContains(t, err, "set MARS_APPLE_AUDIENCES or MARS_APN_BUNDLE_ID")
}
