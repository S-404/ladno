package mock

import "github.com/s-404/ladno/internal/app/entity"

func EnvData() []*entity.Env {
	return []*entity.Env{
		{
			Id:   "env-001",
			Name: "Local",
			Variables: []entity.EnvVariable{
				{Key: "baseUrl", Value: "https://jsonplaceholder.typicode.com", Enabled: true},
				{Key: "token", Value: "local-token", Enabled: true},
			},
		},
		{
			Id:   "env-002",
			Name: "Staging",
			Variables: []entity.EnvVariable{
				{Key: "baseUrl", Value: "https://jsonplaceholder.typicode.com", Enabled: true},
				{Key: "token", Value: "staging-token", Enabled: true},
			},
		},
	}
}
