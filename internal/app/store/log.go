package store

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
)

const maxLogEntries = 500

type ILogStore interface {
	GetItems() *binding.UntypedList
	Append(entry *entity.LogEntry)
	Clear()
	GetItemByIndex(index int) *entity.LogEntry
	GetLogDataItem(item binding.DataItem) *entity.LogEntry
}

type LogStore struct {
	Items binding.UntypedList
}

func NewLogStore() *LogStore {
	return &LogStore{
		Items: binding.NewUntypedList(),
	}
}

func (s *LogStore) GetItems() *binding.UntypedList {
	return &s.Items
}

func (s *LogStore) Append(entry *entity.LogEntry) {
	if entry == nil {
		return
	}
	if entry.Id == "" {
		entry.Id = fmt.Sprintf("log-%d", time.Now().UnixNano())
	}
	if entry.Time.IsZero() {
		entry.Time = time.Now()
	}
	items, _ := s.Items.Get()
	items = append(items, entry)
	if len(items) > maxLogEntries {
		items = items[len(items)-maxLogEntries:]
	}
	_ = s.Items.Set(items)
}

func (s *LogStore) Clear() {
	_ = s.Items.Set(make([]any, 0))
}

func (s *LogStore) GetItemByIndex(index int) *entity.LogEntry {
	item, err := s.Items.GetItem(index)
	if err != nil {
		return nil
	}
	return s.GetLogDataItem(item)
}

func (s *LogStore) GetLogDataItem(item binding.DataItem) *entity.LogEntry {
	val, err := item.(binding.Untyped).Get()
	if err != nil {
		return nil
	}
	entry, ok := val.(*entity.LogEntry)
	if !ok {
		return nil
	}
	return entry
}

func FormatRestLogDetail(resp *entity.RestResponse) string {
	var b strings.Builder

	b.WriteString("── REQUEST ──\n")
	if resp != nil && resp.Request != nil {
		req := resp.Request
		b.WriteString(req.Method)
		b.WriteByte(' ')
		b.WriteString(req.URL)
		b.WriteByte('\n')
		if len(req.Headers) > 0 {
			b.WriteString("\nHeaders:\n")
			writeHeaders(&b, req.Headers)
		}
		if req.Body != "" {
			b.WriteString("\nBody:\n")
			b.WriteString(truncateLogBody(req.Body))
			b.WriteByte('\n')
		}
	} else if resp != nil {
		b.WriteString(resp.Method)
		b.WriteByte(' ')
		b.WriteString(resp.URL)
		b.WriteByte('\n')
	} else {
		b.WriteString("(empty)\n")
	}

	b.WriteString("\n── RESPONSE ──\n")
	if resp == nil {
		b.WriteString("(no response)\n")
		return b.String()
	}
	if resp.Error != "" && resp.StatusCode == 0 {
		b.WriteString("Error: ")
		b.WriteString(resp.Error)
		b.WriteString(fmt.Sprintf("\nDuration: %d ms\n", resp.Duration.Milliseconds()))
		return b.String()
	}
	b.WriteString(fmt.Sprintf("%d %s\n", resp.StatusCode, resp.Status))
	b.WriteString(fmt.Sprintf("Duration: %d ms\n", resp.Duration.Milliseconds()))
	if resp.Error != "" {
		b.WriteString("Error: ")
		b.WriteString(resp.Error)
		b.WriteByte('\n')
	}
	if len(resp.Headers) > 0 {
		b.WriteString("\nHeaders:\n")
		writeHeaders(&b, resp.Headers)
	}
	if resp.Body != "" {
		b.WriteString("\nBody:\n")
		b.WriteString(truncateLogBody(resp.Body))
		b.WriteByte('\n')
	}
	return b.String()
}

func writeHeaders(b *strings.Builder, headers map[string][]string) {
	for k, vals := range headers {
		for _, v := range vals {
			b.WriteString("  ")
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(v)
			b.WriteByte('\n')
		}
	}
}

func truncateLogBody(body string) string {
	const limit = 8 << 10 // 8 KB
	if len(body) <= limit {
		return body
	}
	return body[:limit] + "\n…(truncated)"
}
