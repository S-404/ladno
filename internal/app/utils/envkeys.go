package utils

import (
	"strings"

	"github.com/s-404/ladno/internal/app/entity"
)

// ExtractEnvVarKeys returns unique {{key}} names from input (order of first appearance).
func ExtractEnvVarKeys(input string) []string {
	if input == "" || !strings.Contains(input, "{{") {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	i := 0
	for i < len(input) {
		if i+1 < len(input) && input[i] == '{' && input[i+1] == '{' {
			end := strings.Index(input[i+2:], "}}")
			if end == -1 {
				break
			}
			end += i + 2
			key := strings.TrimSpace(input[i+2 : end])
			if key != "" {
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					out = append(out, key)
				}
			}
			i = end + 2
			continue
		}
		i++
	}
	return out
}

// CollectEnvVarKeys merges {{key}} names from multiple strings.
func CollectEnvVarKeys(parts ...string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range parts {
		for _, k := range ExtractEnvVarKeys(p) {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	return out
}

func appendVarKeys(dst *[]string, seen map[string]struct{}, vars []entity.Variable) {
	for _, v := range vars {
		for _, k := range ExtractEnvVarKeys(v.Key) {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				*dst = append(*dst, k)
			}
		}
		for _, k := range ExtractEnvVarKeys(v.Value) {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				*dst = append(*dst, k)
			}
		}
	}
}

// CollectItemRequestEnvKeys returns all {{key}} tokens used in a request draft.
func CollectItemRequestEnvKeys(req entity.ItemRequest) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(parts ...string) {
		for _, p := range parts {
			for _, k := range ExtractEnvVarKeys(p) {
				if _, ok := seen[k]; ok {
					continue
				}
				seen[k] = struct{}{}
				out = append(out, k)
			}
		}
	}

	add(req.Url.Raw, req.Body)
	appendVarKeys(&out, seen, req.Header)
	appendVarKeys(&out, seen, req.FormData)
	appendVarKeys(&out, seen, req.Url.Variable)
	appendVarKeys(&out, seen, req.Auth.Data)

	if req.Grpc != nil {
		add(req.Grpc.Target, req.Grpc.Method, req.Grpc.Message)
		appendVarKeys(&out, seen, req.Grpc.Metadata)
	}
	if req.Ws != nil {
		add(req.Ws.URL, req.Ws.Message)
		appendVarKeys(&out, seen, req.Ws.Headers)
	}
	if req.Nats != nil {
		add(req.Nats.Subject, req.Nats.Payload)
		appendVarKeys(&out, seen, req.Nats.Headers)
	}
	if req.Kafka != nil {
		add(req.Kafka.Topic, req.Kafka.Key, req.Kafka.Payload)
		appendVarKeys(&out, seen, req.Kafka.Headers)
	}
	return out
}

// CollectCollectionEnvKeys returns {{key}} tokens from collection connection settings.
func CollectCollectionEnvKeys(nats *entity.NatsConnection, kafka *entity.KafkaConnection, auth entity.Auth) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(parts ...string) {
		for _, p := range parts {
			for _, k := range ExtractEnvVarKeys(p) {
				if _, ok := seen[k]; ok {
					continue
				}
				seen[k] = struct{}{}
				out = append(out, k)
			}
		}
	}
	if nats != nil {
		add(nats.Host, nats.Port, nats.Token)
	}
	if kafka != nil {
		add(kafka.Brokers)
	}
	appendVarKeys(&out, seen, auth.Data)
	return out
}
