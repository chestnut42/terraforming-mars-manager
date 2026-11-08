package fcm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"gotest.tools/v3/assert"
)

func newTestService(t *testing.T, handler http.HandlerFunc) *Service {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	assert.NilError(t, err)
	s, err := NewService(Config{BaseURL: u, ProjectId: "proj"}, srv.Client())
	assert.NilError(t, err)
	return s
}

func TestSendNotification(t *testing.T) {
	var gotPath string
	var got map[string]any
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		_, _ = w.Write([]byte(`{"name":"projects/proj/messages/1"}`))
	})

	err := s.SendNotification(context.Background(), "tok", Notification{
		Title: "Mars awaits you!", Body: "1 game is awaiting", Tag: "games", ChannelId: "turns",
	})
	assert.NilError(t, err)
	assert.Equal(t, gotPath, "/v1/projects/proj/messages:send")

	msg := got["message"].(map[string]any)
	assert.Equal(t, msg["token"], "tok")
	assert.Equal(t, msg["notification"].(map[string]any)["title"], "Mars awaits you!")
	android := msg["android"].(map[string]any)
	assert.Equal(t, android["priority"], "high")
	assert.Equal(t, android["notification"].(map[string]any)["tag"], "games")
	assert.Equal(t, android["notification"].(map[string]any)["channel_id"], "turns")
}

func TestSendNotificationUnregistered(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404,"message":"Requested entity was not found.","status":"NOT_FOUND",
			"details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`))
	})
	err := s.SendNotification(context.Background(), "tok", Notification{})
	assert.Assert(t, errors.Is(err, ErrUnregistered), "got %v", err)
}

func TestSendNotificationOtherError(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"denied","status":"PERMISSION_DENIED"}}`))
	})
	err := s.SendNotification(context.Background(), "tok", Notification{})
	assert.Assert(t, err != nil)
	assert.Assert(t, !errors.Is(err, ErrUnregistered))
	assert.ErrorContains(t, err, "PERMISSION_DENIED")
}
