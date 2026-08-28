package entity

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity/constants"
)

func TestResolveAuth(t *testing.T) {
	got := ResolveAuth([]Auth{
		{Type: constants.AuthTypeInherited},
		{Type: constants.AuthTypeInherited},
		{Type: constants.AuthTypeBearer, Data: []Variable{{Key: constants.AuthDataToken, Value: "t"}}},
		{Type: constants.AuthTypeBasic},
	})
	if got.Type != constants.AuthTypeBearer || AuthVar(got.Data, constants.AuthDataToken) != "t" {
		t.Fatalf("got %+v", got)
	}
}

func TestApplyAuthBasicBearerAPIKey(t *testing.T) {
	base := RestRequest{Method: "GET", URL: "http://x", Headers: nil}

	basic := ApplyAuth(base, Auth{
		Type: constants.AuthTypeBasic,
		Data: []Variable{
			{Key: constants.AuthDataUsername, Value: "u"},
			{Key: constants.AuthDataPassword, Value: "p"},
		},
	})
	if AuthVar(basic.Headers, "Authorization") == "" {
		t.Fatal("basic missing Authorization")
	}

	bearer := ApplyAuth(base, Auth{
		Type: constants.AuthTypeBearer,
		Data: []Variable{{Key: constants.AuthDataToken, Value: "abc"}},
	})
	if AuthVar(bearer.Headers, "Authorization") != "Bearer abc" {
		t.Fatalf("bearer: %q", AuthVar(bearer.Headers, "Authorization"))
	}

	customPrefix := ApplyAuth(base, Auth{
		Type: constants.AuthTypeBearer,
		Data: []Variable{
			{Key: constants.AuthDataPrefix, Value: "Token"},
			{Key: constants.AuthDataToken, Value: "xyz"},
		},
	})
	if AuthVar(customPrefix.Headers, "Authorization") != "Token xyz" {
		t.Fatalf("custom prefix: %q", AuthVar(customPrefix.Headers, "Authorization"))
	}

	jsonReq := ApplyAuth(base, Auth{
		Type: constants.AuthTypeJSON,
		Data: []Variable{{Key: constants.AuthDataJSON, Value: `{"token":"x"}`}},
	})
	if AuthVar(jsonReq.Headers, "Authorization") != "" {
		t.Fatal("json auth should not add headers")
	}

	apiH := ApplyAuth(base, Auth{
		Type: constants.AuthTypeAPIKey,
		Data: []Variable{
			{Key: constants.AuthDataKey, Value: "X-Key"},
			{Key: constants.AuthDataValue, Value: "v"},
			{Key: constants.AuthDataAddTo, Value: constants.AuthAddToHeader},
		},
	})
	if AuthVar(apiH.Headers, "X-Key") != "v" {
		t.Fatalf("api header: %+v", apiH.Headers)
	}

	apiB := ApplyAuth(base, Auth{
		Type: constants.AuthTypeAPIKey,
		Data: []Variable{
			{Key: constants.AuthDataKey, Value: "api_key"},
			{Key: constants.AuthDataValue, Value: "v"},
			{Key: constants.AuthDataAddTo, Value: constants.AuthAddToBody},
		},
	})
	if apiB.BodyMode != RestBodyFormData || AuthVar(apiB.FormData, "api_key") != "v" {
		t.Fatalf("api body: mode=%s data=%+v", apiB.BodyMode, apiB.FormData)
	}

	gen := AuthGeneratedHeaders(Auth{
		Type: constants.AuthTypeBearer,
		Data: []Variable{{Key: constants.AuthDataToken, Value: "abc"}},
	})
	if len(gen) != 1 || gen[0].Key != "Authorization" || gen[0].Value != "Bearer abc" {
		t.Fatalf("generated: %+v", gen)
	}
}

func TestSocketIOAuthJSON(t *testing.T) {
	got := SocketIOAuthJSON(Auth{
		Type: constants.AuthTypeJSON,
		Data: []Variable{{Key: constants.AuthDataJSON, Value: `{"token":""}`}},
	}, "legacy")
	if got != `{"token":""}` {
		t.Fatalf("json: %q", got)
	}
	if got := SocketIOAuthJSON(Auth{Type: constants.AuthTypeBearer}, `{"a":1}`); got != "" {
		t.Fatalf("token should not use CONNECT auth: %q", got)
	}
	if got := SocketIOAuthJSON(Auth{Type: constants.AuthTypeNoAuth}, `{"a":1}`); got != `{"a":1}` {
		t.Fatalf("legacy: %q", got)
	}
}

func TestEffectiveSocketIOAuth(t *testing.T) {
	got := EffectiveSocketIOAuth(Auth{Type: constants.AuthTypeInherited}, `{"token":"x"}`)
	if got.Type != constants.AuthTypeJSON || AuthVar(got.Data, constants.AuthDataJSON) != `{"token":"x"}` {
		t.Fatalf("%+v", got)
	}
	got = EffectiveSocketIOAuth(Auth{Type: constants.AuthTypeInherited}, "")
	if got.Type != constants.AuthTypeInherited {
		t.Fatalf("keep inherited: %+v", got)
	}
	got = EffectiveSocketIOAuth(Auth{Type: constants.AuthTypeBearer, Data: []Variable{{Key: constants.AuthDataToken, Value: "t"}}}, `{"x":1}`)
	if got.Type != constants.AuthTypeBearer {
		t.Fatalf("keep bearer: %+v", got)
	}
}
