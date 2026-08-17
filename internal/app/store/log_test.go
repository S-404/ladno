package store

import (
	"testing"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
)

type fixedLimit int

func (l fixedLimit) GetMessageLimit() int { return int(l) }

func TestLogStoreUpdatesIncremental(t *testing.T) {
	s := NewLogStore(fixedLimit(3))
	s.Append(&entity.LogEntry{Message: "a"})
	s.Append(&entity.LogEntry{Message: "b"})
	s.flush()

	select {
	case b := <-s.Updates():
		if len(b.Appended) != 2 || b.DropOldest != 0 {
			t.Fatalf("first batch: %+v", b)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout first batch")
	}

	s.Append(&entity.LogEntry{Message: "c"})
	s.Append(&entity.LogEntry{Message: "d"})
	s.flush()

	select {
	case b := <-s.Updates():
		// before=2, pending=2 → 4, trim to 3 with chunk minDrop=0 (3/6=0) → drop=1
		if len(b.Appended) != 2 {
			t.Fatalf("appended=%d", len(b.Appended))
		}
		if b.DropOldest != 1 {
			t.Fatalf("drop=%d want 1", b.DropOldest)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout second batch")
	}

	s.Clear()
	select {
	case b := <-s.Updates():
		if !b.Cleared {
			t.Fatalf("want cleared: %+v", b)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout clear")
	}
}
