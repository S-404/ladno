package service

import (
	"strings"
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestBuildBodyContentEmptyFormData(t *testing.T) {
	s := NewRestService()
	snap, err := s.BuildSnapshot(entity.RestRequest{
		Method:   "POST",
		URL:      "https://example.com",
		BodyMode: entity.RestBodyFormData,
		FormData: []entity.Variable{{Key: "", Value: ""}, {Key: "", Value: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Body != "" {
		t.Fatalf("expected empty body, got %q", snap.Body)
	}
	for k := range snap.Headers {
		if strings.EqualFold(k, "Content-Type") {
			t.Fatalf("Content-Type should be omitted for empty form-data, got %v", snap.Headers[k])
		}
	}
}

func TestBuildBodyContentFormDataWithFields(t *testing.T) {
	s := NewRestService()
	snap, err := s.BuildSnapshot(entity.RestRequest{
		Method:   "POST",
		URL:      "https://example.com",
		BodyMode: entity.RestBodyFormData,
		FormData: []entity.Variable{{Key: "a", Value: "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Body == "" {
		t.Fatal("expected multipart body")
	}
	ct := snap.Headers["Content-Type"]
	if len(ct) == 0 || !strings.HasPrefix(ct[0], "multipart/form-data") {
		t.Fatalf("want multipart Content-Type, got %v", ct)
	}
}
