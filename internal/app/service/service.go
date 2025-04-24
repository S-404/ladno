package service

import "github.com/s-404/goose/internal/app/repository"

type Service struct {
	Foo IFooService
	Bar IBarService
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Foo: NewFooService(repos.Foo),
		Bar: NewBarService(),
	}
}
