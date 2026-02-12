package entity

import "github.com/s-404/ladno/internal/app/entity/constants"

type Auth struct {
	Type constants.AuthType `json:"type"`
	Data []Variable         `json:"data"`
}
