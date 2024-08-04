package storage

import (
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
