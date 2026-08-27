package service

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
)

func TestFindProtoMethod(t *testing.T) {
	src := `
syntax = "proto3";
package demo;

message Req { string id = 1; }
message Res { string name = 1; }

service User {
  rpc Get (Req) returns (Res);
}
`
	md, err := findProtoMethod([]entity.GrpcProtoFile{{
		Name:    "user.proto",
		Content: src,
	}}, "user.proto", "demo.User/Get")
	if err != nil {
		t.Fatal(err)
	}
	if md.Name() != "Get" {
		t.Fatalf("name=%s", md.Name())
	}
	if md.IsStreamingClient() || md.IsStreamingServer() {
		t.Fatal("expected unary")
	}
}
