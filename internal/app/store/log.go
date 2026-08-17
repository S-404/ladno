package store

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
)

// LogBatch is an incremental update for the logs UI.
type LogBatch struct {
	Appended   []*entity.LogEntry
	DropOldest int                // remove this many rows from the start after append
	Cleared    bool               // wipe the list
	Reset      []*entity.LogEntry // rare: replace entire list (channel overflow)
}

type LogStore struct {
	settings messageLimitSettings

	mu             sync.Mutex
	entries        []*entity.LogEntry
	pending        []*entity.LogEntry
	flushScheduled bool

	updates chan LogBatch
}

// messageLimitSettings is the settings surface LogStore reads.
type messageLimitSettings interface {
	GetMessageLimit() int
}

func NewLogStore(settings messageLimitSettings) *LogStore {
	return &LogStore{
		settings: settings,
		updates:  make(chan LogBatch, 64),
	}
}

func (s *LogStore) messageLimit() int {
	if s.settings == nil {
		return 1000
	}
	return s.settings.GetMessageLimit()
}

// Updates is consumed by the logs panel (single subscriber).
func (s *LogStore) Updates() <-chan LogBatch {
	return s.updates
}

// Append queues a log entry; UI is notified via Updates in coalesced batches.
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

	s.mu.Lock()
	s.pending = append(s.pending, entry)
	limit := s.messageLimit()
	if maxPending := limit * 2; maxPending > 0 && len(s.pending) > maxPending {
		s.pending = s.pending[len(s.pending)-maxPending:]
	}
	schedule := !s.flushScheduled
	if schedule {
		s.flushScheduled = true
	}
	s.mu.Unlock()

	if schedule {
		// Coalesce bursts (~1 frame) without tying store to the UI thread.
		time.AfterFunc(16*time.Millisecond, s.flush)
	}
}

func (s *LogStore) flush() {
	s.mu.Lock()
	pending := s.pending
	s.pending = nil
	s.flushScheduled = false
	if len(pending) == 0 {
		s.mu.Unlock()
		return
	}
	before := len(s.entries)
	s.entries = append(s.entries, pending...)
	s.entries = trimKeepNewest(s.entries, s.messageLimit())
	drop := before + len(pending) - len(s.entries)
	batch := LogBatch{Appended: pending, DropOldest: drop}
	s.mu.Unlock()

	s.emit(batch)
}

func (s *LogStore) TrimToLimit() {
	s.mu.Lock()
	pending := s.pending
	s.pending = nil
	s.flushScheduled = false
	before := len(s.entries)
	s.entries = append(s.entries, pending...)
	s.entries = trimKeepNewest(s.entries, s.messageLimit())
	drop := before + len(pending) - len(s.entries)
	batch := LogBatch{Appended: pending, DropOldest: drop}
	s.mu.Unlock()

	if len(pending) == 0 && drop == 0 {
		return
	}
	s.emit(batch)
}

func (s *LogStore) Clear() {
	s.mu.Lock()
	s.pending = nil
	s.flushScheduled = false
	s.entries = nil
	s.mu.Unlock()
	s.emit(LogBatch{Cleared: true})
}

func (s *LogStore) emit(batch LogBatch) {
	select {
	case s.updates <- batch:
		return
	default:
	}
	// Channel full: drop queued batches and send a full snapshot resync.
	for {
		select {
		case <-s.updates:
		default:
			s.mu.Lock()
			snap := append([]*entity.LogEntry(nil), s.entries...)
			s.mu.Unlock()
			select {
			case s.updates <- LogBatch{Reset: snap}:
			default:
			}
			return
		}
	}
}

func FormatRestLogDetail(resp *entity.RestResponse) string {
	var b strings.Builder

	var snap *entity.RestRequestSnapshot
	if resp != nil {
		snap = resp.Request
		if snap == nil && (resp.Method != "" || resp.URL != "") {
			snap = &entity.RestRequestSnapshot{Method: resp.Method, URL: resp.URL}
		}
	}
	b.WriteString(FormatRestRequestPreview(snap, nil))

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

// FormatRestRequestPreview — текст превью/лога только по REQUEST (как в логах).
func FormatRestRequestPreview(req *entity.RestRequestSnapshot, buildErr error) string {
	var b strings.Builder
	b.WriteString("── REQUEST ──\n")
	if req == nil {
		b.WriteString("(empty)\n")
		if buildErr != nil {
			b.WriteString("\nError: ")
			b.WriteString(buildErr.Error())
			b.WriteByte('\n')
		}
		return b.String()
	}

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
	if buildErr != nil {
		b.WriteString("\nError: ")
		b.WriteString(buildErr.Error())
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
