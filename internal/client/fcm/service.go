package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrUnregistered means the registration token is no longer valid (app
// uninstalled, token rotated) and should be forgotten.
var ErrUnregistered = errors.New("fcm token is unregistered")

// Scope is the OAuth 2.0 scope the HTTP client must be authorized for.
const Scope = "https://www.googleapis.com/auth/firebase.messaging"

type Config struct {
	BaseURL   *url.URL
	ProjectId string
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Notification struct {
	Title string
	Body  string
	// Tag makes a new notification replace an earlier one with the same tag.
	Tag string
	// ChannelId is the Android notification channel.
	ChannelId string
}

// Service sends messages through the FCM HTTP v1 API. The client must add
// OAuth 2.0 credentials (see Scope), e.g. an oauth2.Transport.
type Service struct {
	sendURL string
	client  Client
}

func NewService(cfg Config, client Client) (*Service, error) {
	if cfg.ProjectId == "" {
		return nil, errors.New("fcm project id is empty")
	}
	if cfg.BaseURL == nil {
		return nil, errors.New("fcm base url is empty")
	}
	return &Service{
		sendURL: cfg.BaseURL.JoinPath("v1", "projects", cfg.ProjectId, "messages:send").String(),
		client:  client,
	}, nil
}

type sendRequest struct {
	Message message `json:"message"`
}

type message struct {
	Token        string            `json:"token"`
	Notification *notification     `json:"notification,omitempty"`
	Android      *androidConfig    `json:"android,omitempty"`
	Data         map[string]string `json:"data,omitempty"`
}

type notification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

type androidConfig struct {
	Priority     string               `json:"priority,omitempty"`
	Notification *androidNotification `json:"notification,omitempty"`
}

type androidNotification struct {
	Tag       string `json:"tag,omitempty"`
	ChannelId string `json:"channel_id,omitempty"`
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
		Details []struct {
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

func (s *Service) SendNotification(ctx context.Context, token string, n Notification) error {
	body, err := json.Marshal(sendRequest{Message: message{
		Token:        token,
		Notification: &notification{Title: n.Title, Body: n.Body},
		Android: &androidConfig{
			Priority:     "high",
			Notification: &androidNotification{Tag: n.Tag, ChannelId: n.ChannelId},
		},
	}})
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.sendURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body (status %d): %w", resp.StatusCode, err)
	}
	var errResp errorResponse
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		return fmt.Errorf("failed to send message (status %d): %s", resp.StatusCode, string(respBody))
	}
	for _, d := range errResp.Error.Details {
		if d.ErrorCode == "UNREGISTERED" {
			return ErrUnregistered
		}
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrUnregistered
	}
	return fmt.Errorf("failed to send message (status %d): %s %s",
		resp.StatusCode, errResp.Error.Status, errResp.Error.Message)
}
