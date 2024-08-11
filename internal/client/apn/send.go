package apn

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/httpx"
)

type Alert struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Body     string `json:"body"`
}

type Notification struct {
	Alert Alert  `json:"alert"`
	Badge int    `json:"badge"`
	Sound string `json:"sound"`
}

type fullNotification struct {
	Aps Notification `json:"aps"`
}

func (s *Service) SendNotification(ctx context.Context, device []byte, n Notification) error {
	bodyData, err := json.Marshal(&fullNotification{Aps: n})
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	url := s.baseURL.JoinPath("3", "device", hex.EncodeToString(device)).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	t, err := s.getToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	messageId := uuid.NewString()
	req.Header.Set("Authorization", "Bearer "+t)
	req.Header.Set("apns-id", messageId)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-topic", s.topic)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	if err := httpx.CheckResponse(resp); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}
