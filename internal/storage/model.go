package storage

import (
	"time"

	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type User struct {
	UserId    string
	Nickname  string
	Color     api.PlayerColor
	CreatedAt time.Time
}
