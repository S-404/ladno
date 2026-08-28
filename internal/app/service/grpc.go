package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/s-404/ladno/internal/app/entity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type GrpcService struct{}

func NewGrpcService() *GrpcService {
	return &GrpcService{}
}

func (s *GrpcService) Send(call entity.GrpcCall, cb func(*entity.GrpcResponse)) {
	go func() {
		cb(s.sendSync(call))
	}()
}

func (s *GrpcService) sendSync(call entity.GrpcCall) *entity.GrpcResponse {
	start := time.Now()
	resp := &entity.GrpcResponse{
		Method: call.Method,
		Target: strings.TrimSpace(call.Target),
	}
	fail := func(err error) *entity.GrpcResponse {
		resp.Error = err.Error()
		resp.Duration = time.Since(start)
		return resp
	}
	if strings.TrimSpace(call.Target) == "" {
		return fail(fmt.Errorf("target is empty"))
	}
	if strings.TrimSpace(call.Method) == "" {
		return fail(fmt.Errorf("method is empty"))
	}

	md, err := findProtoMethod(call.ProtoFiles, call.ActiveProto, call.Method)
	if err != nil {
		return fail(err)
	}
	if md.IsStreamingClient() || md.IsStreamingServer() {
		return fail(fmt.Errorf("streaming methods are not supported yet"))
	}

	reqMsg := dynamicpb.NewMessage(md.Input())
	body := strings.TrimSpace(call.Message)
	if body == "" {
		body = "{}"
	}
	unmar := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmar.Unmarshal([]byte(body), reqMsg); err != nil {
		return fail(fmt.Errorf("message json: %w", err))
	}
	respMsg := dynamicpb.NewMessage(md.Output())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, grpcOutgoingMD(call))

	conn, err := grpc.NewClient(resp.Target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fail(err)
	}
	defer func() { _ = conn.Close() }()

	var header, trailer metadata.MD
	path := call.Method
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	err = conn.Invoke(ctx, path, reqMsg, respMsg, grpc.Header(&header), grpc.Trailer(&trailer))
	resp.Duration = time.Since(start)
	resp.Metadata = mergeMD(header, trailer)
	if err != nil {
		st := status.Convert(err)
		resp.Status = st.Code().String()
		resp.StatusCode = int(st.Code())
		resp.Error = st.Message()
		if resp.Error == "" {
			resp.Error = err.Error()
		}
		return resp
	}
	out, mErr := protojson.MarshalOptions{Multiline: true}.Marshal(respMsg)
	if mErr != nil {
		return fail(mErr)
	}
	resp.Status = "OK"
	resp.Body = string(out)
	return resp
}

func grpcOutgoingMD(call entity.GrpcCall) metadata.MD {
	md := metadata.MD{}
	for _, v := range call.Metadata {
		key := strings.TrimSpace(v.Key)
		if key == "" {
			continue
		}
		md.Append(strings.ToLower(key), v.Value)
	}
	for _, h := range entity.AuthGeneratedHeaders(call.Auth) {
		key := strings.ToLower(strings.TrimSpace(h.Key))
		if key == "" {
			continue
		}
		md.Set(key, h.Value)
	}
	return md
}

func mergeMD(parts ...metadata.MD) map[string][]string {
	out := map[string][]string{}
	for _, md := range parts {
		for k, vals := range md {
			out[k] = append(out[k], vals...)
		}
	}
	return out
}

func findProtoMethod(files []entity.GrpcProtoFile, active, method string) (protoreflect.MethodDescriptor, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("import a .proto file")
	}
	compiled, err := compileProtoFiles(files, active)
	if err != nil {
		return nil, err
	}
	full := protoreflect.FullName(strings.Replace(strings.TrimPrefix(strings.TrimSpace(method), "/"), "/", ".", 1))
	desc, err := compiled.AsResolver().FindDescriptorByName(full)
	if err != nil || desc == nil {
		return nil, fmt.Errorf("method %s not found in proto", method)
	}
	md, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not an rpc method", method)
	}
	return md, nil
}

func compileProtoFiles(files []entity.GrpcProtoFile, active string) (linker.Files, error) {
	sources := map[string]string{}
	var names []string
	var importPaths []string
	seenDir := map[string]bool{}
	for _, f := range files {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			name = filepath.Base(f.Path)
		}
		if name == "" {
			continue
		}
		sources[name] = f.Content
		names = append(names, name)
		if dir := filepath.Dir(f.Path); dir != "" && dir != "." && !seenDir[dir] {
			seenDir[dir] = true
			importPaths = append(importPaths, dir)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("import a .proto file")
	}
	root := strings.TrimSpace(active)
	if root == "" || sources[root] == "" {
		root = names[0]
	}
	accessor := protocompile.SourceAccessorFromMap(sources)
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: importPaths,
			Accessor: func(path string) (io.ReadCloser, error) {
				if rc, err := accessor(path); err == nil {
					return rc, nil
				}
				if rc, err := accessor(filepath.Base(path)); err == nil {
					return rc, nil
				}
				return os.Open(path)
			},
		}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	compiled, err := compiler.Compile(ctx, root)
	if err != nil {
		return nil, fmt.Errorf("proto: %w", err)
	}
	return compiled, nil
}
