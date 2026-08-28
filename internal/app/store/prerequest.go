package store

import (
	"fmt"
	"strings"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

// ApplyPreRequest runs preRequest env events before the HTTP send.
// Returns the first error; successful rows are still applied.
func ApplyPreRequest(events []entity.PreRequestEnvEvent, env interface {
	UpsertActiveVar(key, value string) bool
	ClearActiveVar(key string) bool
}) error {
	if env == nil || len(events) == 0 {
		return nil
	}
	var first error
	for _, ev := range events {
		key := strings.TrimSpace(ev.EnvKey)
		if key == "" {
			continue
		}
		action := ev.Action
		if action == "" {
			action = constants.EnvEventActionSet
		}
		switch action {
		case constants.EnvEventActionClear:
			if !env.ClearActiveVar(key) && first == nil {
				first = fmt.Errorf("pre clear %q: no active environment", key)
			}
		case constants.EnvEventActionSet:
			if !env.UpsertActiveVar(key, ev.Value) && first == nil {
				first = fmt.Errorf("pre set %q: no active environment", key)
			}
		}
	}
	return first
}
