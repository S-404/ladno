package utils

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestExtractEnvVarKeys(t *testing.T) {
	got := ExtractEnvVarKeys(`{{baseUrl}}/x/{{id}} and {{baseUrl}}`)
	if len(got) != 2 || got[0] != "baseUrl" || got[1] != "id" {
		t.Fatalf("got=%v", got)
	}
	if ExtractEnvVarKeys("plain") != nil {
		t.Fatal("expected nil")
	}
}

func TestCollectItemRequestEnvKeys(t *testing.T) {
	req := entity.ItemRequest{
		Header: []entity.Variable{{Key: "Authorization", Value: "Bearer {{token}}"}},
		Url:    entity.RequestUrl{Raw: "{{baseUrl}}/posts/:id", Variable: []entity.Variable{{Key: "id", Value: "{{id}}"}}},
		Body:   `{"x":1}`,
	}
	keys := CollectItemRequestEnvKeys(req)
	want := map[string]bool{"baseUrl": true, "token": true, "id": true}
	if len(keys) != len(want) {
		t.Fatalf("keys=%v", keys)
	}
	for _, k := range keys {
		if !want[k] {
			t.Fatalf("unexpected %q in %v", k, keys)
		}
	}
}
