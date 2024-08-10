package main

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewConfig(t *testing.T) {
	_, err := NewConfig()
	assert.NilError(t, err)
}

func TestConfig(t *testing.T) {
	t.Setenv("MARS_LISTEN", ":42")
	t.Setenv("MARS_GAME_URL", "https://example.com/url")
	t.Setenv("MARS_POSTGRES_DSN", "postgres://somedb")
	t.Setenv("MARS_APN_TEAM_ID", "team id")
	t.Setenv("MARS_APN_KEY_ID", "key id")
	t.Setenv("MARS_APN_KEY_FILE", "key file")

	c, err := NewConfig()
	assert.NilError(t, err)
	assert.Equal(t, c.Listen, ":42")
	assert.Equal(t, c.GameURL.String(), "https://example.com/url")
	assert.Equal(t, c.PostgresDSN, "postgres://somedb")
	assert.Equal(t, c.APN.TeamId, "team id")
	assert.Equal(t, c.APN.KeyId, "key id")
	assert.Equal(t, c.APN.KeyFile, "key file")
}
