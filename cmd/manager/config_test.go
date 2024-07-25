package main

import "testing"
import "gotest.tools/v3/assert"

func TestNewConfig(t *testing.T) {
	_, err := NewConfig()
	assert.NilError(t, err)
}
