package service

import "testing"

func TestParseProtoRPCs(t *testing.T) {
	src := `
syntax = "proto3";
package demo.v1;

// service NotAService
service UserService {
  rpc GetUser (GetUserRequest) returns (GetUserResponse);
  rpc Watch (WatchRequest) returns (stream UserEvent);
  rpc Push (stream PushRequest) returns (PushResponse) {
    option deprecated = true;
  }
}

service AdminService {
  rpc Ping (PingRequest) returns (PingResponse);
}
`
	pkg, methods := ParseProtoRPCs(src)
	if pkg != "demo.v1" {
		t.Fatalf("pkg=%q", pkg)
	}
	got := ProtoMethodNames(src)
	want := []string{
		"demo.v1.UserService/GetUser",
		"demo.v1.UserService/Watch",
		"demo.v1.UserService/Push",
		"demo.v1.AdminService/Ping",
	}
	if len(got) != len(want) {
		t.Fatalf("methods=%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
	if !methods[1].ServerStreaming || methods[1].ClientStreaming {
		t.Fatalf("watch streams: %+v", methods[1])
	}
	if !methods[2].ClientStreaming || methods[2].ServerStreaming {
		t.Fatalf("push streams: %+v", methods[2])
	}
}

func TestParseProtoRPCsNoPackage(t *testing.T) {
	src := `service Echo { rpc Send (Msg) returns (Msg); }`
	got := ProtoMethodNames(src)
	if len(got) != 1 || got[0] != "Echo/Send" {
		t.Fatalf("%v", got)
	}
}
