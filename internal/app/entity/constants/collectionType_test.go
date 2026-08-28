package constants

import "testing"

func TestNormalizeCollectionType(t *testing.T) {
	cases := []struct {
		in   CollectionType
		want CollectionType
	}{
		{"", CollectionTypeREST},
		{CollectionTypeREST, CollectionTypeREST},
		{CollectionTypeWS, CollectionTypeWS},
		{CollectionTypeSocketIO, CollectionTypeSocketIO},
		{CollectionTypeGRPC, CollectionTypeGRPC},
		{CollectionTypeNATS, CollectionTypeNATS},
		{CollectionTypeKafka, CollectionTypeKafka},
	}
	for _, tc := range cases {
		if got := NormalizeCollectionType(tc.in); got != tc.want {
			t.Errorf("NormalizeCollectionType(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
	if !IsHTTPCollection(CollectionTypeREST) || !IsHTTPCollection(CollectionTypeWS) ||
		!IsHTTPCollection(CollectionTypeSocketIO) || !IsHTTPCollection(CollectionTypeGRPC) {
		t.Fatal("IsHTTPCollection should cover REST/WS/Socket.IO/gRPC")
	}
	if IsHTTPCollection(CollectionTypeNATS) || IsHTTPCollection(CollectionTypeKafka) {
		t.Fatal("IsHTTPCollection must exclude nats/kafka")
	}
}

func TestRequestKindForCollection(t *testing.T) {
	cases := []struct {
		t    CollectionType
		want RequestKind
	}{
		{CollectionTypeREST, RequestKindREST},
		{CollectionTypeWS, RequestKindWS},
		{CollectionTypeSocketIO, RequestKindSocketIO},
		{CollectionTypeGRPC, RequestKindGRPC},
		{CollectionTypeNATS, RequestKindNATS},
		{CollectionTypeKafka, RequestKindKafka},
	}
	for _, tc := range cases {
		if got := RequestKindForCollection(tc.t); got != tc.want {
			t.Errorf("RequestKindForCollection(%q)=%q want %q", tc.t, got, tc.want)
		}
		if CollectionTypeForKind(tc.want) != tc.t {
			t.Errorf("CollectionTypeForKind(%q) mismatch for %q", tc.want, tc.t)
		}
	}
}
