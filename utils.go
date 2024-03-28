package lib

import "time"

func UtcNow() *time.Time {
	now := time.Now().UTC()
	return &now
}
