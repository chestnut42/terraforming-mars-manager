package main

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewConfig(t *testing.T) {
	_, err := NewConfig()
	assert.NilError(t, err)
}

func TestConfig(t *testing.T) {
	err := os.Setenv("MARS_LISTEN", ":42")
	assert.NilError(t, err)

	err = os.Setenv("MARS_GAME_URL", "https://example.com/url")
	assert.NilError(t, err)

	c, err := NewConfig()
	assert.NilError(t, err)
	assert.Equal(t, c.Listen, ":42")
	assert.Equal(t, c.GameURL.String(), "https://example.com/url")
}
