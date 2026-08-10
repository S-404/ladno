package service

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
)

type IRestService interface {
	Send(req entity.RestRequest, cb func(*entity.RestResponse))
}

type RestService struct {
	client *http.Client
}

func NewRestService() *RestService {
	return &RestService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *RestService) Send(req entity.RestRequest, cb func(*entity.RestResponse)) {
	go func() {
		cb(s.sendSync(req))
	}()
}

func (s *RestService) sendSync(req entity.RestRequest) *entity.RestResponse {
	start := time.Now()

	resolvedURL, err := resolveURL(req.URL, req.PathParams)
	if err != nil {
		return &entity.RestResponse{
			Error:    err.Error(),
			Duration: time.Since(start),
		}
	}

	body, contentType, err := buildBody(req)
	if err != nil {
		return &entity.RestResponse{
			Error:    err.Error(),
			Duration: time.Since(start),
		}
	}

	httpReq, err := http.NewRequest(strings.ToUpper(req.Method), resolvedURL, body)
	if err != nil {
		return &entity.RestResponse{
			Error:    err.Error(),
			Duration: time.Since(start),
		}
	}

	hasContentType := false
	for _, h := range req.Headers {
		if h.Key == "" {
			continue
		}
		httpReq.Header.Add(h.Key, h.Value)
		if strings.EqualFold(h.Key, "Content-Type") {
			hasContentType = true
		}
	}
	if contentType != "" && !hasContentType {
		httpReq.Header.Set("Content-Type", contentType)
	}

	resp, err := s.client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return &entity.RestResponse{
			Error:    err.Error(),
			Duration: duration,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB
	if err != nil {
		return &entity.RestResponse{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Headers:    resp.Header,
			Error:      fmt.Sprintf("read body: %v", err),
			Duration:   duration,
		}
	}

	return &entity.RestResponse{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       string(respBody),
		Duration:   duration,
	}
}

func resolveURL(raw string, pathParams map[string]string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is empty")
	}

	// Подставляем :param в path (не трогая {{vars}} и authority).
	resolved := substitutePathParams(raw, pathParams)

	if !strings.Contains(resolved, "://") {
		resolved = "http://" + resolved
	}

	u, err := url.Parse(resolved)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid url: missing host")
	}
	return u.String(), nil
}

func substitutePathParams(raw string, params map[string]string) string {
	if len(params) == 0 {
		return raw
	}

	path, query, hasQuery := strings.Cut(raw, "?")
	start := 0
	if idx := strings.Index(path, "://"); idx != -1 {
		start = idx + 3
		for start < len(path) && path[start] != '/' {
			start++
		}
	}

	var b strings.Builder
	b.WriteString(path[:start])
	i := start
	for i < len(path) {
		if i+1 < len(path) && path[i] == '{' && path[i+1] == '{' {
			end := strings.Index(path[i:], "}}")
			if end == -1 {
				b.WriteString(path[i:])
				break
			}
			b.WriteString(path[i : i+end+2])
			i += end + 2
			continue
		}
		if path[i] == ':' {
			j := i + 1
			for j < len(path) && isParamChar(path[j]) {
				j++
			}
			if j > i+1 {
				name := path[i+1 : j]
				if val, ok := params[name]; ok {
					b.WriteString(url.PathEscape(val))
				} else {
					b.WriteString(path[i:j])
				}
				i = j
				continue
			}
		}
		b.WriteByte(path[i])
		i++
	}

	if hasQuery {
		return b.String() + "?" + query
	}
	return b.String()
}

func isParamChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-'
}

func buildBody(req entity.RestRequest) (io.Reader, string, error) {
	method := strings.ToUpper(req.Method)
	if method == http.MethodGet || method == http.MethodHead {
		return nil, "", nil
	}

	switch req.BodyMode {
	case entity.RestBodyFormData:
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		for _, row := range req.FormData {
			if row.Key == "" {
				continue
			}
			if err := w.WriteField(row.Key, row.Value); err != nil {
				return nil, "", err
			}
		}
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return &buf, w.FormDataContentType(), nil
	default:
		if req.RawBody == "" {
			return nil, "", nil
		}
		ct := "text/plain"
		trimmed := strings.TrimSpace(req.RawBody)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			ct = "application/json"
		}
		return strings.NewReader(req.RawBody), ct, nil
	}
}
