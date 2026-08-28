package store

import (
	"fmt"
	"strings"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/utils"
)

// ApplyPostRequest runs postRequest env events against response body.
// Returns the first error; successful rows are still applied.
func ApplyPostRequest(body string, events []entity.PostRequestEnvEvent, env interface {
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
				first = fmt.Errorf("clear %q: no active environment", key)
			}
		case constants.EnvEventActionSet:
			path := strings.TrimSpace(ev.JSONPath)
			if path == "" {
				continue
			}
			val, found, err := utils.ExtractDotPath(body, path)
			if err != nil {
				if first == nil {
					first = fmt.Errorf("%s ← %s: %w", key, path, err)
				}
				continue
			}
			if !found {
				continue
			}
			if !env.UpsertActiveVar(key, val) && first == nil {
				first = fmt.Errorf("set %q: no active environment", key)
			}
		}
	}
	return first
}
