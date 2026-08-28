package service

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

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

// BuildSnapshot собирает итоговый вид запроса (URL/headers/body) без отправки.
func (s *RestService) BuildSnapshot(req entity.RestRequest) (*entity.RestRequestSnapshot, error) {
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = http.MethodGet
	}

	resolvedURL, err := resolveURL(req.URL, req.PathParams)
	if err != nil {
		return &entity.RestRequestSnapshot{
			Method: method,
			URL:    strings.TrimSpace(req.URL),
			Body:   previewBody(req),
		}, err
	}

	_, bodyPreview, contentType, err := buildBodyContent(req)
	if err != nil {
		return &entity.RestRequestSnapshot{
			Method: method,
			URL:    resolvedURL,
			Body:   previewBody(req),
		}, err
	}

	headers := make(map[string][]string)
	hasContentType := false
	for _, h := range req.Headers {
		if h.Key == "" {
			continue
		}
		headers[h.Key] = append(headers[h.Key], h.Value)
		if strings.EqualFold(h.Key, "Content-Type") {
			hasContentType = true
		}
	}
	if contentType != "" && !hasContentType {
		headers["Content-Type"] = []string{contentType}
	}

	return &entity.RestRequestSnapshot{
		Method:           method,
		URL:              resolvedURL,
		Headers:          headers,
		Body:             bodyPreview,
		SecretHeaderKeys: append([]string{}, req.SecretHeaderKeys...),
	}, nil
}

func (s *RestService) sendSync(req entity.RestRequest) *entity.RestResponse {
	start := time.Now()
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = http.MethodGet
	}

	resolvedURL, err := resolveURL(req.URL, req.PathParams)
	if err != nil {
		return &entity.RestResponse{
			Method:   method,
			URL:      strings.TrimSpace(req.URL),
			Error:    err.Error(),
			Duration: time.Since(start),
			Request: &entity.RestRequestSnapshot{
				Method: method,
				URL:    strings.TrimSpace(req.URL),
				Body:   previewBody(req),
			},
		}
	}

	// One body build for both wire bytes and Content-Type (multipart boundary must match).
	var bodyReader io.Reader
	var bodyPreview, contentType string
	if method != http.MethodGet && method != http.MethodHead {
		bodyReader, bodyPreview, contentType, err = buildBodyContent(req)
		if err != nil {
			return &entity.RestResponse{
				Method:   method,
				URL:      resolvedURL,
				Error:    err.Error(),
				Duration: time.Since(start),
				Request: &entity.RestRequestSnapshot{
					Method: method,
					URL:    resolvedURL,
					Body:   previewBody(req),
				},
			}
		}
	} else {
		bodyPreview = previewBody(req)
	}

	headers := make(map[string][]string)
	hasContentType := false
	for _, h := range req.Headers {
		if h.Key == "" {
			continue
		}
		headers[h.Key] = append(headers[h.Key], h.Value)
		if strings.EqualFold(h.Key, "Content-Type") {
			hasContentType = true
		}
	}
	if contentType != "" && !hasContentType {
		headers["Content-Type"] = []string{contentType}
	}

	snapshot := &entity.RestRequestSnapshot{
		Method:           method,
		URL:              resolvedURL,
		Headers:          headers,
		Body:             bodyPreview,
		SecretHeaderKeys: append([]string{}, req.SecretHeaderKeys...),
	}

	httpReq, err := http.NewRequest(method, resolvedURL, bodyReader)
	if err != nil {
		return &entity.RestResponse{
			Method:   method,
			URL:      resolvedURL,
			Error:    err.Error(),
			Duration: time.Since(start),
			Request:  snapshot,
		}
	}
	for k, vals := range headers {
		for _, v := range vals {
			httpReq.Header.Add(k, v)
		}
	}

	resp, err := s.client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return &entity.RestResponse{
			Method:   method,
			URL:      resolvedURL,
			Error:    err.Error(),
			Duration: duration,
			Request:  snapshot,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB
	if err != nil {
		return &entity.RestResponse{
			Method:     method,
			URL:        resolvedURL,
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Headers:    resp.Header,
			Error:      fmt.Sprintf("read body: %v", err),
			Duration:   duration,
			Request:    snapshot,
		}
	}

	return &entity.RestResponse{
		Method:     method,
		URL:        resolvedURL,
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Header,
		Body:       string(respBody),
		Duration:   duration,
		Request:    snapshot,
	}
}

func previewBody(req entity.RestRequest) string {
	switch req.BodyMode {
	case entity.RestBodyFormData:
		return formDataPreview(req.FormData)
	case entity.RestBodyURLEncoded:
		return urlEncodedPreview(req.URLEncoded)
	default:
		return req.RawBody
	}
}

func urlEncodedPreview(rows []entity.Variable) string {
	var parts []string
	for _, row := range rows {
		if row.Key == "" {
			continue
		}
		parts = append(parts, row.Key+"="+row.Value)
	}
	return strings.Join(parts, "\n")
}

func formDataPreview(rows []entity.Variable) string {
	var parts []string
	for _, row := range rows {
		if row.Key == "" {
			continue
		}
		if constants.NormalizeFormDataType(row.Type) == constants.FormDataTypeFile {
			path := filesystemPath(row.Value)
			name := filepath.Base(path)
			if name == "" || name == "." {
				name = "(no file)"
			}
			meta := name
			if path != "" {
				if fi, err := os.Stat(path); err == nil {
					meta = fmt.Sprintf("%s, %d bytes", name, fi.Size())
				}
			}
			parts = append(parts, row.Key+"=<file:"+meta+">")
			continue
		}
		parts = append(parts, row.Key+"="+row.Value)
	}
	return strings.Join(parts, "\n")
}

func filesystemPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "file://") {
		if u, err := url.Parse(p); err == nil && u.Path != "" {
			return u.Path
		}
		return strings.TrimPrefix(p, "file://")
	}
	return p
}

func resolveURL(raw string, pathParams map[string]string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is empty")
	}

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

func buildBody(req entity.RestRequest) (io.Reader, string, string, error) {
	method := strings.ToUpper(req.Method)
	if method == http.MethodGet || method == http.MethodHead {
		return nil, "", "", nil
	}
	return buildBodyContent(req)
}

// buildBodyContent собирает тело независимо от method (для preview/snapshot).
// Для form-data preview — человекочитаемый текст; reader — реальный multipart (в т.ч. файлы).
func buildBodyContent(req entity.RestRequest) (io.Reader, string, string, error) {
	switch req.BodyMode {
	case entity.RestBodyFormData:
		fields := make([]entity.Variable, 0, len(req.FormData))
		for _, row := range req.FormData {
			if row.Key == "" {
				continue
			}
			fields = append(fields, row)
		}
		if len(fields) == 0 {
			return nil, "", "", nil
		}
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		for _, row := range fields {
			if constants.NormalizeFormDataType(row.Type) == constants.FormDataTypeFile {
				path := filesystemPath(row.Value)
				if path == "" {
					return nil, "", "", fmt.Errorf("form-data %q: file not selected", row.Key)
				}
				f, err := os.Open(path)
				if err != nil {
					return nil, "", "", fmt.Errorf("form-data %q: open %s: %w", row.Key, path, err)
				}
				part, err := w.CreateFormFile(row.Key, filepath.Base(path))
				if err != nil {
					_ = f.Close()
					return nil, "", "", err
				}
				if _, err := io.Copy(part, f); err != nil {
					_ = f.Close()
					return nil, "", "", fmt.Errorf("form-data %q: read %s: %w", row.Key, path, err)
				}
				_ = f.Close()
				continue
			}
			if err := w.WriteField(row.Key, row.Value); err != nil {
				return nil, "", "", err
			}
		}
		if err := w.Close(); err != nil {
			return nil, "", "", err
		}
		bodyBytes := buf.Bytes()
		return bytes.NewReader(bodyBytes), formDataPreview(fields), w.FormDataContentType(), nil
	case entity.RestBodyURLEncoded:
		vals := url.Values{}
		fields := make([]entity.Variable, 0, len(req.URLEncoded))
		for _, row := range req.URLEncoded {
			if row.Key == "" {
				continue
			}
			vals.Add(row.Key, row.Value)
			fields = append(fields, row)
		}
		if len(fields) == 0 {
			return nil, "", "", nil
		}
		encoded := vals.Encode()
		return strings.NewReader(encoded), urlEncodedPreview(fields), "application/x-www-form-urlencoded", nil
	default:
		if req.RawBody == "" {
			return nil, "", "", nil
		}
		ct := "text/plain"
		trimmed := strings.TrimSpace(req.RawBody)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			ct = "application/json"
		}
		return strings.NewReader(req.RawBody), req.RawBody, ct, nil
	}
}
