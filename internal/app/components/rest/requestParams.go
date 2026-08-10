package rest

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/s-404/ladno/internal/app/components/ui"
)

// NewRequestParams возвращает таб Params с двумя секциями:
//   - Path Variables — параметры (:key) найденные в пути URL
//   - Query Params   — query-строка (?key=value)
//
// Важные инварианты:
//   - URL всегда хранит шаблон с токенами (:id, {{host}}) — path params его НЕ меняют
//   - Path Variables используются только при отправке запроса (подстановка на стороне сервиса)
//   - Query params двусторонние: таблица ↔ URL, с сохранением порядка строк
func NewRequestParams(requestURL binding.String) fyne.CanvasObject {
	var syncing bool

	// ── Path Variables ────────────────────────────────────────────────
	// Таблица только читает имена из URL и хранит значения локально.
	// Запись в URL не производится — значения используются при отправке.

	var pathParamNames []string

	pathTable := ui.NewKVTableReadOnly(nil, nil) // onChange=nil, URL не трогаем

	// ── Query Params ──────────────────────────────────────────────────
	// Двусторонняя синхронизация. Локальный слайс — источник правды для порядка.

	var queryRows []ui.KVRow

	queryTable := ui.NewKVTable(nil, func(rows []ui.KVRow) {
		if syncing || requestURL == nil {
			return
		}
		queryRows = rows
		syncing = true
		defer func() { syncing = false }()
		raw, _ := requestURL.Get()
		requestURL.Set(applyQueryParams(raw, rows))
	})

	// ── Listener на URL ───────────────────────────────────────────────
	// Срабатывает при внешнем изменении URL (пользователь печатает).
	// При записи из queryTable — syncing=true, пропускаем.

	if requestURL != nil {
		requestURL.AddListener(binding.NewDataListener(func() {
			if syncing {
				return
			}
			raw, _ := requestURL.Get()

			// --- path params: обновляем список имён ---
			newNames := extractPathParamNames(raw)
			if !equalStringSlices(newNames, pathParamNames) {
				pathParamNames = newNames
				existingVals := map[string]string{}
				for _, r := range pathTable.GetRows() {
					existingVals[r.Key] = r.Value
				}
				newRows := make([]ui.KVRow, len(newNames))
				for i, name := range newNames {
					newRows[i] = ui.KVRow{
						Enabled: true,
						Key:     name,
						Value:   existingVals[name], // сохраняем введённое значение
					}
				}
				syncing = true
				pathTable.SetRows(newRows)
				syncing = false
			}

			// --- query params: мержим с сохранением порядка и значений ---
			newQueryRows := parseQueryOrdered(raw)
			merged := mergeQueryRows(queryRows, newQueryRows)
			if !equalKVRows(merged, queryRows) {
				queryRows = merged
				syncing = true
				queryTable.SetRows(merged)
				syncing = false
			}
		}))
	}

	// ── GetPathParams — вызывается сервисом при отправке запроса ─────
	// (публичный хелпер, чтобы RestContainer мог получить значения)
	_ = func() map[string]string {
		vals := map[string]string{}
		for _, r := range pathTable.GetRows() {
			if r.Enabled && r.Key != "" && r.Value != "" {
				vals[r.Key] = r.Value
			}
		}
		return vals
	}

	// ── Layout ───────────────────────────────────────────────────────

	pathSection := container.NewBorder(
		sectionLabel("Path Variables"),
		nil, nil, nil,
		pathTable,
	)

	querySection := container.NewBorder(
		sectionLabel("Query Params"),
		nil, nil, nil,
		queryTable,
	)

	return container.NewVBox(pathSection, querySection)
}

// ── URL helpers ───────────────────────────────────────────────────────────────

// skipAuthority возвращает индекс начала пути (после scheme://host:port).
func skipAuthority(s string) int {
	idx := strings.Index(s, "://")
	if idx == -1 {
		return 0
	}
	i := idx + 3
	for i < len(s) && s[i] != '/' {
		i++
	}
	return i
}

// extractPathParamNames возвращает имена :param в порядке появления,
// пропуская authority (scheme://host:port) и {{variables}}.
func extractPathParamNames(rawURL string) []string {
	path, _, _ := strings.Cut(rawURL, "?")
	start := skipAuthority(path)

	var names []string
	seen := map[string]bool{}
	i := start
	for i < len(path) {
		if i+1 < len(path) && path[i] == '{' && path[i+1] == '{' {
			end := strings.Index(path[i:], "}}")
			if end == -1 {
				break
			}
			i += end + 2
			continue
		}
		if path[i] == ':' {
			j := i + 1
			for j < len(path) && isParamChar(path[j]) {
				j++
			}
			if j > i+1 {
				name := path[i+1 : j]
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
				i = j
				continue
			}
		}
		i++
	}
	return names
}

// applyQueryParams пересобирает query-часть URL из строк таблицы.
// Порядок строк сохраняется. Disabled и строки без key — пропускаются.
func applyQueryParams(rawURL string, rows []ui.KVRow) string {
	base, _, _ := strings.Cut(rawURL, "?")

	var parts []string
	for _, r := range rows {
		if r.Enabled && r.Key != "" {
			parts = append(parts, r.Key+"="+r.Value)
		}
	}

	if len(parts) == 0 {
		return base
	}
	return base + "?" + strings.Join(parts, "&")
}

// parseQueryOrdered парсит query вручную, сохраняя порядок ключей.
func parseQueryOrdered(rawURL string) []ui.KVRow {
	_, query, found := strings.Cut(rawURL, "?")
	if !found || query == "" {
		return nil
	}

	var rows []ui.KVRow
	for _, part := range strings.Split(query, "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		rows = append(rows, ui.KVRow{Enabled: true, Key: k, Value: v})
	}
	return rows
}

// mergeQueryRows объединяет старые строки (источник порядка) с новыми (из URL).
//   - Порядок определяется old
//   - Value берётся из new (URL — источник правды для значений)
//   - Enabled сохраняется из old
//   - Строки из new, которых нет в old, добавляются в конец
//   - Строки из old, которых нет в new, удаляются
func mergeQueryRows(old, new []ui.KVRow) []ui.KVRow {
	newCount := map[string]int{}
	for _, r := range new {
		newCount[r.Key]++
	}
	oldCount := map[string]int{}
	for _, r := range old {
		oldCount[r.Key]++
	}

	// Строим map: key → очередь значений из new
	newVals := map[string][]string{}
	for _, r := range new {
		newVals[r.Key] = append(newVals[r.Key], r.Value)
	}

	// Если наборы ключей совпадают — обновляем только Value, порядок и Enabled из old
	if equalCountMaps(newCount, oldCount) {
		usedIdx := map[string]int{}
		result := make([]ui.KVRow, len(old))
		for i, r := range old {
			idx := usedIdx[r.Key]
			val := r.Value
			if idx < len(newVals[r.Key]) {
				val = newVals[r.Key][idx]
			}
			result[i] = ui.KVRow{Enabled: r.Enabled, Key: r.Key, Value: val}
			usedIdx[r.Key]++
		}
		return result
	}

	// Наборы ключей различаются — перестраиваем
	used := map[string]int{}
	var result []ui.KVRow

	for _, r := range old {
		if newCount[r.Key] > used[r.Key] {
			idx := used[r.Key]
			val := r.Value
			if idx < len(newVals[r.Key]) {
				val = newVals[r.Key][idx]
			}
			result = append(result, ui.KVRow{Enabled: r.Enabled, Key: r.Key, Value: val})
			used[r.Key]++
		}
	}

	// Добавляем ключи которых не было в old
	addedFromNew := map[string]bool{}
	for _, r := range new {
		if oldCount[r.Key] == 0 && !addedFromNew[r.Key] {
			result = append(result, r)
			addedFromNew[r.Key] = true
		}
	}

	return result
}

// ── UI helpers ────────────────────────────────────────────────────────────────

func sectionLabel(text string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// ── generic helpers ───────────────────────────────────────────────────────────

func isParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-'
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalKVRows(a, b []ui.KVRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalCountMaps(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
