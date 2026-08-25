package service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
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
	if snap.Body != "a=1" {
		t.Fatalf("preview body=%q", snap.Body)
	}
	ct := snap.Headers["Content-Type"]
	if len(ct) == 0 || !strings.HasPrefix(ct[0], "multipart/form-data") {
		t.Fatalf("want multipart Content-Type, got %v", ct)
	}
}

func TestBuildBodyContentFormDataFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/hello.txt"
	if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewRestService()
	req := entity.RestRequest{
		Method:   "POST",
		URL:      "https://example.com",
		BodyMode: entity.RestBodyFormData,
		FormData: []entity.Variable{
			{Key: "doc", Value: path, Type: constants.FormDataTypeFile},
			{Key: "note", Value: "x", Type: constants.FormDataTypeText},
		},
	}
	snap, err := s.BuildSnapshot(req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snap.Body, "doc=<file:hello.txt") || !strings.Contains(snap.Body, "note=x") {
		t.Fatalf("preview=%q", snap.Body)
	}
	reader, preview, ct, err := buildBodyContent(req)
	if err != nil {
		t.Fatal(err)
	}
	if preview != snap.Body {
		t.Fatalf("preview mismatch %q vs %q", preview, snap.Body)
	}
	if !strings.HasPrefix(ct, "multipart/form-data") {
		t.Fatalf("ct=%q", ct)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("hi")) || !bytes.Contains(raw, []byte(`filename="hello.txt"`)) {
		t.Fatalf("multipart missing file contents: %q", raw)
	}
	// Content-Type boundary must appear in the body.
	const prefix = "multipart/form-data; boundary="
	if !strings.HasPrefix(ct, prefix) {
		t.Fatalf("ct=%q", ct)
	}
	boundary := strings.TrimPrefix(ct, prefix)
	if !bytes.Contains(raw, []byte("--"+boundary)) {
		t.Fatalf("body missing boundary %q", boundary)
	}
}

func TestSendFormDataFileUploadsContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upload.bin")
	payload := []byte("file-bytes-xyz")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	var gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	s := NewRestService()
	s.client = srv.Client()
	resp := s.sendSync(entity.RestRequest{
		Method:   "POST",
		URL:      srv.URL,
		BodyMode: entity.RestBodyFormData,
		FormData: []entity.Variable{
			{Key: "f", Value: path, Type: constants.FormDataTypeFile},
		},
	})
	if resp.Error != "" {
		t.Fatal(resp.Error)
	}
	if !strings.HasPrefix(gotCT, "multipart/form-data; boundary=") {
		t.Fatalf("ct=%q", gotCT)
	}
	boundary := strings.TrimPrefix(gotCT, "multipart/form-data; boundary=")
	if !bytes.Contains(gotBody, []byte("--"+boundary)) {
		t.Fatalf("boundary mismatch ct=%q body=%q", gotCT, gotBody)
	}
	if !bytes.Contains(gotBody, payload) {
		t.Fatalf("file contents not uploaded: %q", gotBody)
	}
	if !bytes.Contains(gotBody, []byte(`filename="upload.bin"`)) {
		t.Fatalf("missing filename disposition: %q", gotBody)
	}
	// Must be a file part, not a plain text field with the path.
	if bytes.Contains(gotBody, []byte(path)) {
		t.Fatalf("raw path sent as text field: %q", gotBody)
	}
}
