package store

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/utils"
)

// restService is the HTTP surface RestStore needs.
type restService interface {
	Send(req entity.RestRequest, cb func(*entity.RestResponse))
	BuildSnapshot(req entity.RestRequest) (*entity.RestRequestSnapshot, error)
}

// restEnvVars is the env lookup RestStore needs.
type restEnvVars interface {
	ActiveVariables() map[string]string
	UpsertActiveVar(key, value string) bool
	ClearActiveVar(key string) bool
}

// restLog is the log append surface RestStore needs.
type restLog interface {
	Append(entry *entity.LogEntry)
}

// restCookies is the cookie jar surface RestStore needs.
type restCookies interface {
	AbsorbResponse(requestURL string, headers map[string][]string)
	CookieHeaderForURL(rawURL string) string
}

type RestStore struct {
	Draft       binding.Untyped
	Response    binding.Untyped
	IsSending   binding.Bool
	restService restService
	envStore    restEnvVars
	logStore    restLog
	cookies     restCookies
}

func NewRestStore(svc restService, envStore restEnvVars, logStore restLog, cookies restCookies) *RestStore {
	return &RestStore{
		Draft:       binding.NewUntyped(),
		Response:    binding.NewUntyped(),
		IsSending:   binding.NewBool(),
		restService: svc,
		envStore:    envStore,
		logStore:    logStore,
		cookies:     cookies,
	}
}

func (s *RestStore) GetDraft() binding.Untyped {
	return s.Draft
}

func (s *RestStore) SetDraft(draft entity.RestDraft) {
	// pointer: RestDraft содержит map/slice и несравним для binding.Untyped
	d := draft
	_ = s.Draft.Set(&d)
}

func (s *RestStore) GetIsSending() *binding.Bool {
	return &s.IsSending
}

func (s *RestStore) GetResponse() binding.Untyped {
	return s.Response
}

func (s *RestStore) ClearResponse() {
	s.Response.Set(nil)
}

func (s *RestStore) Preview(req entity.RestRequest, showSecrets bool) string {
	req = s.prepareRequest(req)
	snap, err := s.restService.BuildSnapshot(req)
	return FormatRestRequestPreview(snap, err, showSecrets)
}

func (s *RestStore) Send(req entity.RestRequest) {
	if sending, _ := s.IsSending.Get(); sending {
		fmt.Println("rest: already sending")
		return
	}
	s.IsSending.Set(true)
	s.Response.Set(nil)

	var scriptErr error
	if len(req.PreRequest) > 0 {
		if err := ApplyPreRequest(req.PreRequest, s.envStore); err != nil {
			scriptErr = err
		}
	}

	req = s.prepareRequest(req)

	s.restService.Send(req, func(resp *entity.RestResponse) {
		fyne.Do(func() {
			if resp != nil && s.cookies != nil && resp.Headers != nil {
				s.cookies.AbsorbResponse(resp.URL, resp.Headers)
			}
			if resp != nil && len(req.PostRequest) > 0 {
				if err := ApplyPostRequest(resp.Body, req.PostRequest, s.envStore); err != nil {
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
			s.logRest(resp)
		})
	})
}

func (s *RestStore) prepareRequest(req entity.RestRequest) entity.RestRequest {
	if s.envStore != nil {
		vars := s.envStore.ActiveVariables()
		req = applyEnvVars(req, vars)
		req.Auth = applyEnvToAuth(req.Auth, vars)
	}
	req = entity.ApplyAuth(req, req.Auth)
	req = s.applyCookies(req)
	return req
}

func (s *RestStore) applyCookies(req entity.RestRequest) entity.RestRequest {
	if s.cookies == nil || entity.HasHeaderKey(req.Headers, "Cookie") {
		return req
	}
	val := s.cookies.CookieHeaderForURL(req.URL)
	if val == "" {
		return req
	}
	req.Headers = append(req.Headers, entity.Variable{Key: "Cookie", Value: val})
	return req
}

func applyEnvToAuth(auth entity.Auth, vars map[string]string) entity.Auth {
	if len(vars) == 0 || len(auth.Data) == 0 {
		return auth
	}
	data := make([]entity.Variable, len(auth.Data))
	for i, d := range auth.Data {
		data[i] = entity.Variable{
			Key:   utils.SubstituteEnvVars(d.Key, vars),
			Value: utils.SubstituteEnvVars(d.Value, vars),
			Type:  d.Type,
		}
	}
	auth.Data = data
	return auth
}

func (s *RestStore) logRest(resp *entity.RestResponse) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:       "rest",
		Message:    formatRestResult(resp),
		Detail:     FormatRestLogDetail(resp),
		StatusCode: statusCodeOf(resp),
		IsError:    isRestError(resp),
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

func statusCodeOf(resp *entity.RestResponse) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func isRestError(resp *entity.RestResponse) bool {
	if resp == nil {
		return true
	}
	return resp.Error != "" && resp.StatusCode == 0
}

func formatRestResult(resp *entity.RestResponse) string {
	if resp == nil {
		return "no response"
	}
	method := resp.Method
	if method == "" {
		method = "?"
	}
	url := resp.URL
	if url == "" {
		url = "?"
	}
	dur := resp.Duration.Milliseconds()
	if resp.Error != "" && resp.StatusCode == 0 {
		return fmt.Sprintf("ERR %s %s (%d ms): %s", method, url, dur, resp.Error)
	}
	if resp.Error != "" {
		return fmt.Sprintf("%d %s %s (%d ms): %s", resp.StatusCode, method, url, dur, resp.Error)
	}
	return fmt.Sprintf("%d %s %s (%d ms)", resp.StatusCode, method, url, dur)
}

func applyEnvVars(req entity.RestRequest, vars map[string]string) entity.RestRequest {
	if len(vars) == 0 {
		return req
	}

	req.URL = utils.SubstituteEnvVars(req.URL, vars)
	req.RawBody = utils.SubstituteEnvVars(req.RawBody, vars)

	if len(req.PathParams) > 0 {
		params := make(map[string]string, len(req.PathParams))
		for k, v := range req.PathParams {
			params[k] = utils.SubstituteEnvVars(v, vars)
		}
		req.PathParams = params
	}

	if len(req.Headers) > 0 {
		headers := make([]entity.Variable, len(req.Headers))
		for i, h := range req.Headers {
			headers[i] = entity.Variable{
				Key:   utils.SubstituteEnvVars(h.Key, vars),
				Value: utils.SubstituteEnvVars(h.Value, vars),
				Type:  h.Type,
			}
		}
		req.Headers = headers
	}

	if len(req.FormData) > 0 {
		form := make([]entity.Variable, len(req.FormData))
		for i, f := range req.FormData {
			form[i] = entity.Variable{
				Key:   utils.SubstituteEnvVars(f.Key, vars),
				Value: utils.SubstituteEnvVars(f.Value, vars),
				Type:  f.Type,
			}
		}
		req.FormData = form
	}

	return req
}
