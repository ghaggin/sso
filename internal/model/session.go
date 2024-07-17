package model

import "time"

type Session struct {
	UID            string
	AuthValid      bool
	AuthExpiration time.Time
}
