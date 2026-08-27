package entity

import "time"

// GrpcCall is the resolved snapshot used to invoke a unary RPC.
type GrpcCall struct {
	Target      string
	Method      string // package.Service/Method
	Message     string
	Metadata    []Variable
	Auth        Auth
	ProtoFiles  []GrpcProtoFile
	ActiveProto string
	PreRequest  []PreRequestEnvEvent
	PostRequest []PostRequestEnvEvent
}

// GrpcResponse is the result of a unary RPC.
type GrpcResponse struct {
	Method      string
	Target      string
	Status      string
	StatusCode  int
	Metadata    map[string][]string
	Body        string
	Duration    time.Duration
	Error       string
	ScriptError string
}
