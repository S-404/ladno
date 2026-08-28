package utils

import "testing"

func TestSubstituteEnvVars(t *testing.T) {
	vars := map[string]string{
		"baseUrl": "https://api.example.com",
		"token":   "abc",
	}

	got := SubstituteEnvVars("{{baseUrl}}/posts/{{id}}", vars)
	want := "https://api.example.com/posts/{{id}}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	got = SubstituteEnvVars("Bearer {{token}}", vars)
	if got != "Bearer abc" {
		t.Fatalf("got %q", got)
	}

	got = SubstituteEnvVars("plain", vars)
	if got != "plain" {
		t.Fatalf("got %q", got)
	}
}
