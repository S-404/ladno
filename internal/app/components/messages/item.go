package messages

import (
	"strings"
	"time"
)

const (
	DirIn  = "in"
	DirOut = "out"
)

type Item struct {
	Time time.Time
	Dir  string
	Body string
}

func DirArrow(dir string) string {
	if dir == DirIn {
		return "←"
	}
	return "→"
}

func FormatTime(t time.Time) string {
	return t.Format("15:04:05.000")
}

func FormatItem(it Item) string {
	ts := FormatTime(it.Time)
	arrow := DirArrow(it.Dir)
	if strings.TrimSpace(it.Body) == "" {
		return "[" + ts + "] " + arrow
	}
	return "[" + ts + "] " + arrow + "\n" + it.Body
}

func FilterItems(items []Item, q string) []Item {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return items
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if itemMatches(it, q) {
			out = append(out, it)
		}
	}
	return out
}

func VisibleItems(items []Item, filter string, all bool) []Item {
	items = FilterItems(items, filter)
	if !all && len(items) > 0 {
		return items[len(items)-1:]
	}
	return items
}

func itemMatches(it Item, q string) bool {
	if strings.Contains(strings.ToLower(it.Body), q) {
		return true
	}
	if strings.Contains(FormatTime(it.Time), q) {
		return true
	}
	if strings.Contains(DirArrow(it.Dir), q) {
		return true
	}
	return it.Dir != "" && strings.Contains(strings.ToLower(it.Dir), q)
}
