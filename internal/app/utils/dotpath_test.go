package utils

import "testing"

func TestExtractDotPath(t *testing.T) {
	body := `{"token":"abc","data":{"user":{"id":42}},"items":[{"name":"a"}],"nilish":null}`

	cases := []struct {
		path    string
		want    string
		found   bool
		wantErr bool
	}{
		{"token", "abc", true, false},
		{"data.user.id", "42", true, false},
		{"items.0.name", "a", true, false},
		{"missing", "", false, true},
		{"data?.missing?.x", "", false, false},
		{"missing?.x", "", false, false},
		{"nilish?.x", "", false, false},
		{"nilish.x", "", false, true},
		{"data.user?.id", "42", true, false},
		{"", "", false, true},
		{"items.5.name", "", false, true},
		{"items?.5?.name", "", false, false},
	}
	for _, tc := range cases {
		got, found, err := ExtractDotPath(body, tc.path)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%q: want err", tc.path)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tc.path, err)
		}
		if found != tc.found || got != tc.want {
			t.Fatalf("%q: got (%q,%v), want (%q,%v)", tc.path, got, found, tc.want, tc.found)
		}
	}
}

func TestExtractDotPathBadJSON(t *testing.T) {
	_, _, err := ExtractDotPath("not-json", "a")
	if err == nil {
		t.Fatal("want json error")
	}
}
