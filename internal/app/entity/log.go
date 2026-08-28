package entity

import "time"

// LogEntry — запись в нижнем лог-панели.
type LogEntry struct {
	Id         string    `json:"id"`
	Time       time.Time `json:"time"`
	Kind       string    `json:"kind"` // rest, nats, system, …
	Message    string    `json:"message"`
	Detail     string    `json:"detail"` // plain fallback
	StatusCode int       `json:"statusCode"`
	IsError    bool      `json:"isError"`
	Highlight  bool      `json:"highlight"` // входящие NATS (subscribe/request)
}

// RestRequestSnapshot — итоговый HTTP-запрос после env/path подстановок.
type RestRequestSnapshot struct {
	Method           string              `json:"method"`
	URL              string              `json:"url"`
	Headers          map[string][]string `json:"headers"`
	Body             string              `json:"body"`
	SecretHeaderKeys []string            `json:"secretHeaderKeys,omitempty"`
}
