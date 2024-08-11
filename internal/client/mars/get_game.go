package mars

import (
	"context"
	"encoding/json"
	"fmt"
)

type GetGamePlayer struct {
	Id    string
	Score int
}

type GetGameModel struct {
	HasFinished bool
	Players     []GetGamePlayer
}

type GetGameResponse struct {
	Game GetGameModel
	Raw  map[string]any
}

func (s *Service) GetGame(ctx context.Context, spectator string) (*GetGameResponse, error) {
	return nil, nil
}

func readResponse(data []byte) (*GetGameResponse, error) {
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw response: %w", err)
	}

	var resp getGameResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get game response: %w", err)
	}

	players := make([]GetGamePlayer, len(resp.Players))
	for i, p := range resp.Players {
		players[i] = GetGamePlayer{
			Id:    p.Id,
			Score: p.VPBreakdown.Total,
		}
	}
	return &GetGameResponse{
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
	VPBreakdown getGameVictoryPointsBreakdown `json:"victoryPointsBreakdown"`
}

type getGameVictoryPointsBreakdown struct {
	Total int `json:"total"`
}
