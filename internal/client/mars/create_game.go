package mars

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type NewPlayer struct {
	Id    string
	Name  string
	Color storage.Color
}

type CreateGame struct {
	Players []NewPlayer
}

type CreateGameResponse struct {
	Id          string
	SpectatorId string
	Players     []NewPlayer
	PurgeDate   time.Time
}

func (s *Service) CreateGame(ctx context.Context, game CreateGame) (CreateGameResponse, error) {
	req := defaultCreateGame()
	req.Players = requestPlayers(game.Players)

	reqData, err := json.Marshal(req)
	if err != nil {
		return CreateGameResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut,
		s.baseURL.JoinPath("game").String(), bytes.NewReader(reqData))
	if err != nil {
		return CreateGameResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	httpResp, err := s.client.Do(httpReq)
	if err != nil {
		return CreateGameResponse{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer httpResp.Body.Close()

	var resp createGameResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return CreateGameResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	respPlayers := make([]NewPlayer, len(resp.Players))
	for i, p := range resp.Players {
		respPlayers[i] = NewPlayer{
			Id:    p.Id,
			Name:  p.Name,
			Color: storage.Color(p.Color),
		}
	}
	return CreateGameResponse{
		Id:          resp.Id,
		SpectatorId: resp.SpectatorId,
		Players:     respPlayers,
		PurgeDate:   time.UnixMilli(resp.PurgeDateMs),
	}, nil
}

func requestPlayers(players []NewPlayer) []newPlayer {
	leftColors := make(map[storage.Color]struct{})
	for _, c := range allColors {
		leftColors[c] = struct{}{}
	}

	conflictingPlayers := make([]int, 0)
	for i, player := range players {
		c := player.Color
		if _, ok := leftColors[c]; ok {
			delete(leftColors, c)
		} else {
			conflictingPlayers = append(conflictingPlayers, i)
		}
	}

	for _, conflictIdx := range conflictingPlayers {
		for _, c := range allColors {
			if _, ok := leftColors[c]; ok {
				players[conflictIdx].Color = c
				delete(leftColors, c)
				break
			}
		}
	}

	newPlayers := make([]newPlayer, len(players))
	for i, p := range players {
		newPlayers[i] = newPlayer{
			Name:     p.Name,
			Color:    string(p.Color),
			Beginner: false,
			Handicap: 0,
			First:    false,
		}
	}

	firstPlayer := rand.N(len(players))
	newPlayers[firstPlayer].First = true
	return newPlayers
}

func defaultCreateGame() createGame {
	return createGame{
		CorporateEra:              true,
		Prelude:                   true,
		ShowOtherPlayersVP:        true,
		VenusNext:                 true,
		CustomCorporationsList:    make([]any, 0),
		CustomColoniesList:        make([]any, 0),
		CustomPreludes:            make([]any, 0),
		BannedCards:               make([]any, 0),
		IncludedCards:             make([]any, 0),
		Board:                     "tharsis",
		Seed:                      rand.Float32(),
		PoliticalAgendasExtension: "Standard",
		UndoOption:                true,
		ShowTimers:                true,
		IncludeVenusMA:            true,
		StartingCorporations:      3,
		PreludeDraftVariant:       true,
		RandomMA:                  "No randomization",
		CustomCeos:                make([]any, 0),
		StartingCeos:              3,
	}
}

type newPlayer struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Beginner bool   `json:"beginner"`
	Handicap int    `json:"handicap"`
	First    bool   `json:"first"`
}

type createGame struct {
	Players           []newPlayer `json:"players"`
	Prelude           bool        `json:"prelude"`
	VenusNext         bool        `json:"venusNext"`
	Colonies          bool        `json:"colonies"`
	Turmoil           bool        `json:"turmoil"`
	Board             string      `json:"board"`
	Seed              float32     `json:"seed"`
	RandomFirstPlayer bool        `json:"randomFirstPlayer"`

	// Configuration
	UndoOption         bool `json:"undoOption"`
	ShowTimers         bool `json:"showTimers"`
	FastModeOption     bool `json:"fastModeOption"`
	ShowOtherPlayersVP bool `json:"showOtherPlayersVP"`

	// Extensions
	CorporateEra                     bool   `json:"corporateEra"`
	Prelude2Expansion                bool   `json:"prelude2Expansion"`
	PromoCardsOption                 bool   `json:"promoCardsOption"`
	CommunityCardsOption             bool   `json:"communityCardsOption"`
	AresExtension                    bool   `json:"aresExtension"`
	PoliticalAgendasExtension        string `json:"politicalAgendasExtension"`
	SolarPhaseOption                 bool   `json:"solarPhaseOption"`
	RemoveNegativeGlobalEventsOption bool   `json:"removeNegativeGlobalEventsOption"`
	IncludeVenusMA                   bool   `json:"includeVenusMA"`
	MoonExpansion                    bool   `json:"moonExpansion"`
	PathfindersExpansion             bool   `json:"pathfindersExpansion"`
	CeoExtension                     bool   `json:"ceoExtension"`

	// Variants
	DraftVariant                 bool   `json:"draftVariant"`
	InitialDraft                 bool   `json:"initialDraft"`
	PreludeDraftVariant          bool   `json:"preludeDraftVariant"`
	StartingCorporations         int    `json:"startingCorporations"`
	ShuffleMapOption             bool   `json:"shuffleMapOption"`
	RandomMA                     string `json:"randomMA"`
	IncludeFanMA                 bool   `json:"includeFanMA"`
	SoloTR                       bool   `json:"soloTR"`
	CustomCorporationsList       []any  `json:"customCorporationsList"`
	BannedCards                  []any  `json:"bannedCards"`
	IncludedCards                []any  `json:"includedCards"`
	CustomColoniesList           []any  `json:"customColoniesList"`
	CustomPreludes               []any  `json:"customPreludes"`
	RequiresMoonTrackCompletion  bool   `json:"requiresMoonTrackCompletion"`
	RequiresVenusTrackCompletion bool   `json:"requiresVenusTrackCompletion"`
	MoonStandardProjectVariant   bool   `json:"moonStandardProjectVariant"`
	MoonStandardProjectVariant1  bool   `json:"moonStandardProjectVariant1"`
	AltVenusBoard                bool   `json:"altVenusBoard"`
	EscapeVelocityMode           bool   `json:"escapeVelocityMode"`
	TwoCorpsVariant              bool   `json:"twoCorpsVariant"`
	CustomCeos                   []any
	StartingCeos                 int
	StarWarsExpansion            bool `json:"starWarsExpansion"`
	UnderworldExpansion          bool `json:"underworldExpansion"`
}

type newPlayerResponse struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type createGameResponse struct {
	Id          string              `json:"id"`
	SpectatorId string              `json:"spectatorId"`
	Players     []newPlayerResponse `json:"players"`
	PurgeDateMs int64               `json:"expectedPurgeTimeMs"`
}

var allColors = []storage.Color{
	storage.ColorBlue,
	storage.ColorRed,
	storage.ColorYellow,
	storage.ColorGreen,
	storage.ColorBlack,
	storage.ColorPurple,
	storage.ColorOrange,
	storage.ColorPink,
	storage.ColorBronze,
}
