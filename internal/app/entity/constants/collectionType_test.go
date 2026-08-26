package constants

import "testing"

func TestNormalizeCollectionType(t *testing.T) {
	cases := []struct {
		in   CollectionType
		want CollectionType
	}{
		{CollectionTypeHTTP, CollectionTypeHTTP},
		{CollectionTypeREST, CollectionTypeHTTP},
		{CollectionTypeGRPC, CollectionTypeHTTP},
		{CollectionTypeWS, CollectionTypeHTTP},
		{"", CollectionTypeHTTP},
		{CollectionTypeNATS, CollectionTypeNATS},
		{CollectionTypeKafka, CollectionTypeKafka},
	}
	for _, tc := range cases {
		if got := NormalizeCollectionType(tc.in); got != tc.want {
			t.Errorf("NormalizeCollectionType(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if !IsHTTPCollection(CollectionTypeREST) || !IsHTTPCollection(CollectionTypeGRPC) || !IsHTTPCollection(CollectionTypeWS) {
		t.Fatal("legacy types should be HTTP collections")
	}
	if IsHTTPCollection(CollectionTypeNATS) || IsHTTPCollection(CollectionTypeKafka) {
		t.Fatal("nats/kafka should not be HTTP collections")
	}
}
