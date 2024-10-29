package app

import (
	"context"
	"errors"
	"math/rand/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/chestnut42/terraforming-mars-manager/internal/auth"
	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

func (s *Service) CreateGame(ctx context.Context, req *api.CreateGame_Request) (*api.CreateGame_Response, error) {
	users, err := s.getPlayers(ctx, req.GetPlayers())
	if err != nil {
		return nil, err
	}

	if err := s.game.CreateGame(ctx, users, mars.GameSettings{
		Board:        mars.BoardTharsis,
		CorporateEra: true,
		Prelude:      true,
		VenusNext:    true,
		SolarPhase:   false,
	}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &api.CreateGame_Response{}, nil
}

func (s *Service) CreateGameV2(ctx context.Context, req *api.CreateGameV2_Request) (*api.CreateGameV2_Response, error) {
	users, err := s.getPlayers(ctx, req.GetPlayers())
	if err != nil {
		return nil, err
	}

	if err := s.game.CreateGame(ctx, users, mars.GameSettings{
		Board:        boardFromAPIV2(req.GetBoard()),
		CorporateEra: req.GetCorporateEra(),
		Prelude:      req.GetPrelude(),
		VenusNext:    req.GetVenusNext(),
		SolarPhase:   req.GetSolarPhase(),
	}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &api.CreateGameV2_Response{}, nil
}

func (s *Service) getPlayers(ctx context.Context, players []string) ([]*storage.User, error) {
	thisUser, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	// Check if all users are unique
	if !isUnique(players) {
		return nil, status.Error(codes.InvalidArgument, "players are not unique")
	}

	users := make([]*storage.User, len(players))
	for i, player := range players {
		u, err := s.storage.GetUserByNickname(ctx, player)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return nil, status.Error(codes.NotFound, "user not found")
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		users[i] = u
	}

	hasThisUser := false
	for _, u := range users {
		if u.UserId == thisUser.Id {
			hasThisUser = true
		}
	}
	if !hasThisUser {
		u, err := s.storage.GetUserById(ctx, thisUser.Id)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		users = append(users, u)
	}
	if len(users) > 5 {
		return nil, status.Errorf(codes.InvalidArgument, "too many players: %d", len(users))
	}
	return users, nil
}

func (s *Service) GetGames(ctx context.Context, _ *api.GetGames_Request) (*api.GetGames_Response, error) {
	thisUser, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	games, err := s.game.GetUserGames(ctx, thisUser.Id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &api.GetGames_Response{Games: []*api.Game{}}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	apiGames := make([]*api.Game, len(games))
	for i, g := range games {
		st := api.GameStatus_GAME_STATUS_IN_PROGRESS
		if g.AwaitsInput {
			st = api.GameStatus_GAME_STATUS_AWAITS_INPUT
		}
		if g.HasFinished {
			st = api.GameStatus_GAME_STATUS_FINISHED
		}

		apiGames[i] = &api.Game{
			PlayUrl:      g.PlayURL,
			CreatedAt:    timestamppb.New(g.CreatedAt),
			ExpiresAt:    timestamppb.New(g.ExpiresAt),
			PlayersCount: int32(g.PlayersCount),
			AwaitsInput:  g.AwaitsInput,
			Status:       st,
		}
	}
	return &api.GetGames_Response{Games: apiGames}, nil
}

func isUnique(str []string) bool {
	m := make(map[string]struct{})
	for _, v := range str {
		m[v] = struct{}{}
	}
	return len(m) == len(str)
}

func boardFromAPIV2(board api.CreateGameV2_Board) mars.Board {
	switch board {
	case api.CreateGameV2_THARSIS:
		return mars.BoardTharsis
	case api.CreateGameV2_HELLAS:
		return mars.BoardHellas
	case api.CreateGameV2_ELYSIUM:
		return mars.BoardElysium
	default:
		return mars.AllBoards[rand.IntN(len(mars.AllBoards))]
	}
}
