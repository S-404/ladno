package store

import "testing"

func TestTrimKeepNewestChunkDrop(t *testing.T) {
	// Reference NatsClientStore: if len > 600 → slice(100) keeps ~501.
	in := make([]int, 601)
	for i := range in {
		in[i] = i
	}
	out := trimKeepNewest(in, 600)
	if len(out) != 501 {
		t.Fatalf("len=%d want 501", len(out))
	}
	if out[0] != 100 || out[len(out)-1] != 600 {
		t.Fatalf("range %d..%d", out[0], out[len(out)-1])
	}
}

func TestTrimKeepNewestExactCap(t *testing.T) {
	in := make([]int, 2000)
	for i := range in {
		in[i] = i
	}
	out := trimKeepNewest(in, 1000)
	if len(out) != 1000 {
		t.Fatalf("len=%d want 1000", len(out))
	}
	if out[0] != 1000 || out[999] != 1999 {
		t.Fatalf("range %d..%d", out[0], out[999])
	}
}

func TestTrimKeepNewestNoOp(t *testing.T) {
	in := []int{1, 2, 3}
	out := trimKeepNewest(in, 10)
	if len(out) != 3 {
		t.Fatal(out)
	}
}
