package state

import "time"

type Session struct {
	RefreshToken string
	Expiry       time.Time
}
