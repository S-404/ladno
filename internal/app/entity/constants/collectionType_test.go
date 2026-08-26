package constants

import "testing"

func TestNormalizeCollectionType(t *testing.T) {
	cases := []struct {
		in   CollectionType
		want CollectionType
	}{
		{CollectionTypeHTTP, CollectionTypeHTTP},
		{"", CollectionTypeHTTP},
		{CollectionTypeNATS, CollectionTypeNATS},
		{CollectionTypeKafka, CollectionTypeKafka},
	}
	for _, tc := range cases {
		if got := NormalizeCollectionType(tc.in); got != tc.want {
			t.Errorf("NormalizeCollectionType(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if !IsHTTPCollection(CollectionTypeHTTP) || IsHTTPCollection(CollectionTypeNATS) || IsHTTPCollection(CollectionTypeKafka) {
		t.Fatal("IsHTTPCollection")
	}
}
