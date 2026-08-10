package store

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/service"
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
}

func NewRestStore(restService service.IRestService) *RestStore {
	return &RestStore{
		Draft:       binding.NewUntyped(),
		Response:    binding.NewUntyped(),
		IsSending:   binding.NewBool(),
		restService: restService,
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

	s.restService.Send(req, func(resp *entity.RestResponse) {
		fyne.Do(func() {
			_ = s.Response.Set(resp)
			_ = s.IsSending.Set(false)
		})
	})
}
