package storage

import "time"

type User struct {
	UserId    string
	Nickname  string
	CreatedAt time.Time
}
