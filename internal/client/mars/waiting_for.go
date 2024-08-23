package mars

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/httpx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type WaitingForRequest struct {
	PlayerId string
}

type WaitingForResponse struct {
	Colors []storage.Color
}

func (s *Service) WaitingFor(ctx context.Context, req WaitingForRequest) (WaitingForResponse, error) {
	reqUrl := *s.cfg.BaseURL
	reqUrl.Path = path.Join(reqUrl.Path, "api/waitingfor")
	v := url.Values{}
	v.Set("id", req.PlayerId)
	reqUrl.RawQuery = v.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl.String(), nil)
	if err != nil {
		return WaitingForResponse{}, fmt.Errorf("failed to create http request: %w", err)
	}

	httpResp, err := s.client.Do(httpReq)
	if err != nil {
		return WaitingForResponse{}, fmt.Errorf("failed to send http request: %w", err)
	}
	defer httpResp.Body.Close()

	if err := httpx.CheckResponse(httpResp); err != nil {
		return WaitingForResponse{}, fmt.Errorf("invalid http response: %w", err)
	}

	var resp waitingForResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return WaitingForResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	respColors := make([]storage.Color, len(resp.WaitingFor))
	for i, color := range resp.WaitingFor {
		respColors[i] = storage.Color(color)
	}
	return WaitingForResponse{Colors: respColors}, nil
}

type waitingForResponse struct {
	Result     string   `json:"result"`
	WaitingFor []string `json:"waitingFor"`
}
