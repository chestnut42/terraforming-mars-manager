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

type User struct {
	UserId      string
	Nickname    string
	Color       Color
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
	Color    Color
}

type SentNotification struct {
	ActiveGames int `json:"ag"`
}

type SentNotificationUpdater func(ctx context.Context, sn SentNotification) (SentNotification, error)

func (sn *SentNotification) Value() (driver.Value, error) {
	return json.Marshal(sn)
}

func (sn *SentNotification) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &sn)
}
