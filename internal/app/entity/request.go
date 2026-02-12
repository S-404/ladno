package entity

import "github.com/s-404/ladno/internal/app/entity/constants"

type RequestUrl struct {
	Raw      string     `json:"raw"`
	Variable []Variable `json:"variable"`
}

type ItemRequest struct {
	Method constants.RequestMethod `json:"method"`
	Header []Variable              `json:"header"`
	Auth   Auth                    `json:"auth"`
	Event  Event                   `json:"event"`
	Url    RequestUrl              `json:"url"`
}
