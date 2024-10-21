package app

import (
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

var toAPIColors = map[storage.Color]api.PlayerColor{
	storage.ColorBlue:   api.PlayerColor_BLUE,
	storage.ColorRed:    api.PlayerColor_RED,
	storage.ColorYellow: api.PlayerColor_YELLOW,
	storage.ColorGreen:  api.PlayerColor_GREEN,
	storage.ColorBlack:  api.PlayerColor_BLACK,
	storage.ColorPurple: api.PlayerColor_PURPLE,
	storage.ColorOrange: api.PlayerColor_ORANGE,
	storage.ColorPink:   api.PlayerColor_PINK,
	storage.ColorBronze: api.PlayerColor_BRONZE,
}

var fromAPIColors = map[api.PlayerColor]storage.Color{
	api.PlayerColor_BLUE:   storage.ColorBlue,
	api.PlayerColor_RED:    storage.ColorRed,
	api.PlayerColor_YELLOW: storage.ColorYellow,
	api.PlayerColor_GREEN:  storage.ColorGreen,
	api.PlayerColor_BLACK:  storage.ColorBlack,
	api.PlayerColor_PURPLE: storage.ColorPurple,
	api.PlayerColor_ORANGE: storage.ColorOrange,
	api.PlayerColor_PINK:   storage.ColorPink,
	api.PlayerColor_BRONZE: storage.ColorBronze,
}

func userToAPI(user *storage.User) *api.User {
	return &api.User{
		Id:        user.UserId,
		Nickname:  user.Nickname,
		Color:     toAPIColors[user.Color],
		CreatedAt: timestamppb.New(user.CreatedAt),
		Elo:       int32(user.Elo),
	}
}

func fromAPIColor(color api.PlayerColor) (storage.Color, error) {
	c, ok := fromAPIColors[color]
	if !ok {
		return "", fmt.Errorf("unknown color: %d", color)
	}
	return c, nil
}
