package store

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/utils"
)

type grpcService interface {
	Send(call entity.GrpcCall, cb func(*entity.GrpcResponse))
}

type grpcEnvVars interface {
	ActiveVariables() map[string]string
	UpsertActiveVar(key, value string) bool
	ClearActiveVar(key string) bool
}

type grpcLog interface {
	Append(entry *entity.LogEntry)
}

type GrpcStore struct {
	Response    binding.Untyped
	IsSending   binding.Bool
	grpcService grpcService
	envStore    grpcEnvVars
	logStore    grpcLog
}

func NewGrpcStore(svc grpcService, envStore grpcEnvVars, logStore grpcLog) *GrpcStore {
	return &GrpcStore{
		Response:    binding.NewUntyped(),
		IsSending:   binding.NewBool(),
		grpcService: svc,
		envStore:    envStore,
		logStore:    logStore,
	}
}

func (s *GrpcStore) GetIsSending() *binding.Bool {
	return &s.IsSending
}

func (s *GrpcStore) GetResponse() binding.Untyped {
	return s.Response
}

func (s *GrpcStore) ClearResponse() {
	_ = s.Response.Set(nil)
}

func (s *GrpcStore) Send(call entity.GrpcCall) {
	if sending, _ := s.IsSending.Get(); sending {
		return
	}
	_ = s.IsSending.Set(true)
	_ = s.Response.Set(nil)

	var scriptErr error
	if len(call.PreRequest) > 0 {
		if err := ApplyPreRequest(call.PreRequest, s.envStore); err != nil {
			scriptErr = err
		}
	}
	call = s.prepare(call)
	s.grpcService.Send(call, func(resp *entity.GrpcResponse) {
		fyne.Do(func() {
			if resp != nil && len(call.PostRequest) > 0 {
				if err := ApplyPostRequest(resp.Body, call.PostRequest, s.envStore); err != nil {
					if scriptErr == nil {
						scriptErr = err
					} else {
						scriptErr = fmt.Errorf("%w; %v", scriptErr, err)
					}
				}
			}
			if resp != nil && scriptErr != nil {
				resp.ScriptError = scriptErr.Error()
			}
			_ = s.Response.Set(resp)
			_ = s.IsSending.Set(false)
			s.logGrpc(resp)
		})
	})
}

func (s *GrpcStore) prepare(call entity.GrpcCall) entity.GrpcCall {
	if s.envStore == nil {
		return call
	}
	vars := s.envStore.ActiveVariables()
	if len(vars) == 0 {
		return call
	}
	call.Target = utils.SubstituteEnvVars(call.Target, vars)
	call.Method = utils.SubstituteEnvVars(call.Method, vars)
	call.Message = utils.SubstituteEnvVars(call.Message, vars)
	call.Metadata = substituteVarList(call.Metadata, vars, true)
	call.Auth = applyEnvToAuth(call.Auth, vars)
	return call
}

func (s *GrpcStore) logGrpc(resp *entity.GrpcResponse) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:       "grpc",
		Message:    formatGrpcResult(resp),
		Detail:     formatGrpcDetail(resp),
		StatusCode: grpcLogStatus(resp),
		IsError:    grpcIsError(resp),
	})
	if resp != nil && resp.ScriptError != "" {
		s.logStore.Append(&entity.LogEntry{
			Kind:    "script",
			Message: "Script error: " + resp.ScriptError,
			Detail:  resp.ScriptError,
			IsError: true,
		})
	}
}

func formatGrpcResult(resp *entity.GrpcResponse) string {
	if resp == nil {
		return "no response"
	}
	method := resp.Method
	if method == "" {
		method = "?"
	}
	if resp.Error != "" && resp.StatusCode == 0 && resp.Status == "" {
		return "ERR gRPC " + method + ": " + resp.Error
	}
	st := resp.Status
	if st == "" {
		st = "OK"
	}
	if resp.Error != "" {
		return fmt.Sprintf("%s gRPC %s %s: %s", st, method, resp.Target, resp.Error)
	}
	return fmt.Sprintf("%s gRPC %s %s (%d ms)", st, method, resp.Target, resp.Duration.Milliseconds())
}

func formatGrpcDetail(resp *entity.GrpcResponse) string {
	if resp == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("── gRPC ──\n")
	b.WriteString("Target: " + resp.Target + "\n")
	b.WriteString("Method: " + resp.Method + "\n")
	b.WriteString("Status: " + resp.Status + "\n")
	b.WriteString(fmt.Sprintf("Duration: %d ms\n", resp.Duration.Milliseconds()))
	if resp.Error != "" {
		b.WriteString("Error: " + resp.Error + "\n")
	}
	if resp.Body != "" {
		b.WriteString("\n")
		b.WriteString(resp.Body)
		b.WriteByte('\n')
	}
	return b.String()
}

func grpcIsError(resp *entity.GrpcResponse) bool {
	if resp == nil {
		return true
	}
	return resp.Error != ""
}

func grpcLogStatus(resp *entity.GrpcResponse) int {
	if resp == nil || resp.Error != "" {
		return 0
	}
	if resp.Status == "OK" {
		return 200
	}
	return resp.StatusCode
}
