package app

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

func userToAPI(user *storage.User) *api.User {
	return &api.User{
		Id:        user.UserId,
		Nickname:  user.Nickname,
		Color:     user.Color,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}
