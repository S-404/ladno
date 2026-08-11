package service

import "github.com/s-404/ladno/internal/app/repository"

type Service struct {
	Env       IEnvService
	Kafka     IKafkaService
	Nats      INatsService
	Rest      IRestService
	Workspace IWorkspaceService
}

func NewService(repos *repository.Repository) *Service {
	return &Service{
		Env:       NewEnvService(repos.Env),
		Kafka:     NewKafkaService(),
		Nats:      NewNatsService(),
		Rest:      NewRestService(),
		Workspace: newWorkspaceService(repos.Workspace),
	}
}
