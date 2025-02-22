package storage

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Color string

const (
	ColorBlue   Color = "blue"
	ColorRed    Color = "red"
	ColorYellow Color = "yellow"
	ColorGreen  Color = "green"
	ColorBlack  Color = "black"
	ColorPurple Color = "purple"
	ColorOrange Color = "orange"
	ColorPink   Color = "pink"
	ColorBronze Color = "bronze"
)

type DeviceTokenType string

const (
	DeviceTokenTypeSandbox    DeviceTokenType = "sdbx"
	DeviceTokenTypeProduction DeviceTokenType = "prod"
)

type UserType string

const (
	UserTypeBlank  UserType = "blank"  // Just created users receive this type
	UserTypeActive UserType = "active" // Users that has changed their username and are ready to play
)

type User struct {
	UserId          string
	Nickname        string
	Color           Color
	CreatedAt       time.Time
	DeviceToken     []byte
	DeviceTokenType DeviceTokenType
	LastIp          string
	Type            UserType
	Elo             int64
}

type Game struct {
	GameId      string
	SpectatorId string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	FinishedAt  *time.Time
	Players     []Player
	GameResults *GameResults
}

type Player struct {
	UserId   string
	PlayerId string
	Color    Color
}

type SentNotification struct {
	ActiveGames int `json:"ag"`
}

type UserNotificationState struct {
	DeviceToken      []byte
	DeviceTokenType  DeviceTokenType
	SentNotification SentNotification
}

type SentNotificationUpdater func(ctx context.Context, state UserNotificationState) (UserNotificationState, error)

type EloResultsPlayer struct {
	PlayerId string
	UserId   string
	OldElo   int64
	NewElo   int64
}

type EloResultsPair struct {
	LeftPlayerId    string
	RightPlayerId   string
	LeftPlayerElo   int64
	RightPlayerElo  int64
	LeftEloDelta    int64
	LeftPlayerScore float64
}

type EloResults struct {
	Pairs   []EloResultsPair
	Players []EloResultsPlayer
}

type EloStateUser struct {
	UserId string
	Elo    int64
}

type EloUpdateState struct {
	Game  Game
	Users []EloStateUser
}

type EloUpdater func(ctx context.Context, state EloUpdateState) (EloResults, error)

func (sn *SentNotification) Value() (driver.Value, error) {
	return json.Marshal(sn)
}

func (sn *SentNotification) Scan(value interface{}) error {
	if value == nil {
		*sn = SentNotification{}
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &sn)
}

type GameResults struct {
	Raw map[string]any
}

func (r *GameResults) Value() (driver.Value, error) {
	v, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshal GameResults failed: %v", err)
	}
	return v, nil
}

func (r *GameResults) Scan(value interface{}) error {
	if value == nil {
		*r = GameResults{}
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	return json.Unmarshal(b, r)
}

func (er *EloResults) Value() (driver.Value, error) {
	return json.Marshal(er)
}

func (er *EloResults) Scan(value interface{}) error {
	if value == nil {
		*er = EloResults{}
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &er)
}
