package game

import (
	"context"
	"fmt"
	"time"
)

type UserGame struct {
	PlayURL      string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	PlayersCount int
	AwaitsInput  bool
}

func (s *Service) GetUserGames(ctx context.Context, userId string) ([]*UserGame, error) {
	return nil, fmt.Errorf("unimplemented")
}
