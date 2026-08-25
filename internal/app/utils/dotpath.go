package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// pathSeg — сегмент пути: a.b?.c → [{a,false},{b,true},{c,false}] для a.b?.c
// (optional относится к доступу этого сегмента: ?.c значит «если parent null — skip»).
type pathSeg struct {
	Key      string
	Optional bool
}

// ExtractDotPath читает значение из JSON body по пути с точками и optional chaining (?).
// Примеры: "token", "data.user.id", "data?.user?.token", "items.0.name".
//
// found=false, err=nil — путь оборвался на optional-сегменте (тихий skip).
// found=false, err!=nil — обязательный путь не найден / невалидный JSON / пустой path.
func ExtractDotPath(body, path string) (value string, found bool, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false, fmt.Errorf("empty path")
	}
	segs, err := parseDotPath(path)
	if err != nil {
		return "", false, err
	}
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		if segs[0].Optional {
			return "", false, nil
		}
		return "", false, fmt.Errorf("empty response body")
	}
	var root any
	if err := json.Unmarshal([]byte(trimmed), &root); err != nil {
		return "", false, fmt.Errorf("response is not JSON: %w", err)
	}
	cur := root
	for i, seg := range segs {
		if isNullish(cur) {
			if seg.Optional {
				return "", false, nil
			}
			return "", false, fmt.Errorf("path %q: cannot access %q on null", path, seg.Key)
		}
		next, ok, miss := dig(cur, seg.Key)
		if miss {
			if i == len(segs)-1 {
				if seg.Optional {
					return "", false, nil
				}
				return "", false, fmt.Errorf("path %q: missing %q", path, seg.Key)
			}
			// Intermediate miss → nullish; next segment's optional (if any) decides.
			cur = nil
			continue
		}
		if !ok {
			if seg.Optional {
				return "", false, nil
			}
			return "", false, fmt.Errorf("path %q: cannot access %q", path, seg.Key)
		}
		cur = next
	}
	if cur == nil {
		return "", true, nil
	}
	s, err := jsonValueToString(cur)
	if err != nil {
		return "", false, err
	}
	return s, true, nil
}

func isNullish(v any) bool {
	return v == nil
}

func parseDotPath(path string) ([]pathSeg, error) {
	var segs []pathSeg
	i := 0
	for i < len(path) {
		optional := false
		if strings.HasPrefix(path[i:], "?.") {
			optional = true
			i += 2
		} else if i > 0 && path[i] == '.' {
			i++
		} else if i > 0 {
			return nil, fmt.Errorf("invalid path %q", path)
		}
		if i >= len(path) {
			return nil, fmt.Errorf("invalid path %q", path)
		}
		if strings.HasPrefix(path[i:], "?.") || path[i] == '.' {
			return nil, fmt.Errorf("invalid path %q", path)
		}
		j := i
		for j < len(path) {
			if path[j] == '.' {
				break
			}
			if j+1 < len(path) && path[j] == '?' && path[j+1] == '.' {
				break
			}
			j++
		}
		key := path[i:j]
		if key == "" || strings.Contains(key, "?") {
			return nil, fmt.Errorf("invalid path segment in %q", path)
		}
		segs = append(segs, pathSeg{Key: key, Optional: optional})
		i = j
	}
	if len(segs) == 0 {
		return nil, fmt.Errorf("empty path")
	}
	return segs, nil
}

// dig returns (value, ok, missing).
// missing=true when key/index absent; ok=false when type cannot be indexed.
func dig(cur any, key string) (any, bool, bool) {
	switch v := cur.(type) {
	case map[string]any:
		val, ok := v[key]
		if !ok {
			return nil, false, true
		}
		return val, true, false
	case []any:
		idx, err := strconv.Atoi(key)
		if err != nil {
			return nil, false, false
		}
		if idx < 0 || idx >= len(v) {
			return nil, false, true
		}
		return v[idx], true, false
	default:
		return nil, false, false
	}
}

func jsonValueToString(v any) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10), nil
		}
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(t), nil
	case nil:
		return "", nil
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
