package apn

import (
	_ "embed"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

//go:embed test_key.p8
var testKeyData []byte

func TestNewService(t *testing.T) {
	s, err := NewService(Config{
		TeamId:  "team id",
		KeyId:   "key id",
		KeyData: testKeyData,
	}, nil)
	assert.NilError(t, err)
	assert.Assert(t, s.teamId == "team id")
	assert.Assert(t, s.key != nil)

	kid, ok := s.key.Get("kid")
	assert.Assert(t, ok)
	assert.Assert(t, kid == "key id")

	now := time.Unix(1723300000, 0)
	s.now = func() time.Time { return now }
	token, err := s.getToken()
	assert.NilError(t, err)
	assert.Assert(t, strings.HasPrefix(token, "eyJhbGciOiJFUzI1NiIsImtpZCI6ImtleSBpZCIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE3MjMzMDAwMDAsImlzcyI6InRlYW0gaWQifQ."), "received: %s", token)
}
