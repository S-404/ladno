package store

// trimKeepNewest drops oldest items when over limit.
// Like NatsClientStore (if len > 600 → slice(100)): when over limit, remove at
// least ~limit/6 so continuous floods don't trim-by-one on every append.
func trimKeepNewest[T any](items []T, limit int) []T {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	drop := len(items) - limit
	if minDrop := limit / 6; minDrop > 0 && drop < minDrop {
		drop = minDrop
	}
	if drop > len(items)-1 {
		drop = len(items) - limit
	}
	if drop <= 0 {
		return items[len(items)-limit:]
	}
	out := items[drop:]
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

func trimKeepNewestAny(items []any, limit int) []any {
	return trimKeepNewest(items, limit)
}
