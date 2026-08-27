package store

import "time"

// StreamMessage is a protocol-agnostic row for the shared Messages pane.
type StreamMessage struct {
	Time time.Time
	Dir  string // "in" | "out"
	Body string
}
