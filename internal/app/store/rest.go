package store

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/utils"
)

type IRestStore interface {
	GetDraft() binding.Untyped
	SetDraft(draft entity.RestDraft)
	GetIsSending() *binding.Bool
	GetResponse() binding.Untyped
	Send(req entity.RestRequest)
	ClearResponse()
}

type RestStore struct {
	Draft       binding.Untyped
	Response    binding.Untyped
	IsSending   binding.Bool
	restService service.IRestService
	envStore    IEnvStore
}

func NewRestStore(restService service.IRestService, envStore IEnvStore) *RestStore {
	return &RestStore{
		Draft:       binding.NewUntyped(),
		Response:    binding.NewUntyped(),
		IsSending:   binding.NewBool(),
		restService: restService,
		envStore:    envStore,
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

func (s *RestStore) Send(req entity.RestRequest) {
	if sending, _ := s.IsSending.Get(); sending {
		fmt.Println("rest: already sending")
		return
	}
	s.IsSending.Set(true)
	s.Response.Set(nil)

	if s.envStore != nil {
		req = applyEnvVars(req, s.envStore.ActiveVariables())
	}

	s.restService.Send(req, func(resp *entity.RestResponse) {
		fyne.Do(func() {
			_ = s.Response.Set(resp)
			_ = s.IsSending.Set(false)
		})
	})
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
