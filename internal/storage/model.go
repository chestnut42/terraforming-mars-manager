package storage

import (
	"time"

	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type User struct {
	UserId      string
	Nickname    string
	Color       api.PlayerColor
	CreatedAt   time.Time
	DeviceToken []byte
}

type Game struct {
	GameId      string
	SpectatorId string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Players     []*Player
}

type Player struct {
	UserId   string
	PlayerId string
}
