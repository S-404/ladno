package utils

import "strings"

// SubstituteEnvVars заменяет токены {{key}} значениями из vars.
// Неизвестный ключ оставляется как есть.
func SubstituteEnvVars(input string, vars map[string]string) string {
	if input == "" || len(vars) == 0 || !strings.Contains(input, "{{") {
		return input
	}

	var b strings.Builder
	b.Grow(len(input))
	i := 0
	for i < len(input) {
		if i+1 < len(input) && input[i] == '{' && input[i+1] == '{' {
			end := strings.Index(input[i+2:], "}}")
			if end == -1 {
				b.WriteString(input[i:])
				break
			}
			end += i + 2
			key := strings.TrimSpace(input[i+2 : end])
			if val, ok := vars[key]; ok {
				b.WriteString(val)
			} else {
				b.WriteString(input[i : end+2])
			}
			i = end + 2
			continue
		}
		b.WriteByte(input[i])
		i++
	}
	return b.String()
}
