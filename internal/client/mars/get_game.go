package mars

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/httpx"
)

type GetGamePlayer struct {
	Id          string
	MegaCredits int
	Score       int
}

type GetGameModel struct {
	HasFinished bool
	Players     []GetGamePlayer
}

type GetGameRequest struct {
	SpectatorId string
}

type GetGameResponse struct {
	Game GetGameModel
	Raw  map[string]any
}

func (s *Service) GetGame(ctx context.Context, req GetGameRequest) (GetGameResponse, error) {
	reqUrl := *s.cfg.BaseURL
	reqUrl.Path = path.Join(reqUrl.Path, "api/spectator")
	v := url.Values{}
	v.Set("id", req.SpectatorId)
	reqUrl.RawQuery = v.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl.String(), nil)
	if err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to create http request: %w", err)
	}

	httpResp, err := s.client.Do(httpReq)
	if err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to send http request: %w", err)
	}
	defer httpResp.Body.Close() //nolint:errcheck

	if err := httpx.CheckResponse(httpResp); err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to check http response: %w", err)
	}

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}

	resp, err := readResponse(data)
	if err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to read response struct: %w", err)
	}
	return resp, nil
}

func GetGameResponseFromRaw(raw map[string]any) (GetGameResponse, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to marshal raw response: %w", err)
	}

	resp, err := readResponse(data)
	if err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to read response: %w", err)
	}
	return resp, nil
}

func readResponse(data []byte) (GetGameResponse, error) {
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to unmarshal raw response: %w", err)
	}

	var resp getGameResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return GetGameResponse{}, fmt.Errorf("failed to unmarshal get game response: %w", err)
	}

	players := make([]GetGamePlayer, len(resp.Players))
	for i, p := range resp.Players {
		players[i] = GetGamePlayer{
			Id:          p.Id,
			MegaCredits: p.MegaCredits,
			Score:       p.VPBreakdown.Total,
		}
	}
	return GetGameResponse{
		Game: GetGameModel{
			HasFinished: resp.Game.Phase == "end",
			Players:     players,
		},
		Raw: raw,
	}, nil
}

type getGameResponse struct {
	Game    getGameGame     `json:"game"`
	Players []getGamePlayer `json:"players"`
}

type getGameGame struct {
	Phase string `json:"phase"`
}

type getGamePlayer struct {
	Id          string                        `json:"id"`
	MegaCredits int                           `json:"megaCredits"`
	VPBreakdown getGameVictoryPointsBreakdown `json:"victoryPointsBreakdown"`
}

type getGameVictoryPointsBreakdown struct {
	Total int `json:"total"`
}
