package mock

import (
	"github.com/google/uuid"
	"github.com/s-404/goose/internal/app/entity"
)

// MockWorkspaceData возвращает мок данные для workspace
func MockWorkspaceData() []*entity.Workspace {
	return []*entity.Workspace{
		{
			Id:               "ws-001",
			Title:            "Основное рабочее пространство",
			ConnectionConfig: "postgresql://localhost:5432/main",
			CollectionItems: []entity.CollectionItem{
				{
					Id:    uuid.NewString(),
					Title: "Пользователи",
					Auth: entity.Auth{
						Type: "JWT",
						Data: []entity.AuthData{
							{
								Key:   "Token",
								Type:  "string",
								Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
							},
						},
					},
					Event: entity.Event{
						PreRequest: []entity.PreRequestEnvEvent{
							{
								EnvKey: "token2",
								Action: entity.EnvEventActionClear,
							},
						},
						PostRequest: []entity.PostRequestEnvEvent{
							{
								EnvKey:   "token",
								Action:   entity.EnvEventActionSet,
								JSONPath: "data.token",
							},
						},
					},
				},
				{
					Id:    "col-002",
					Title: "Заказы",
					Auth: entity.Auth{
						Type: "basic",
						Data: []entity.AuthData{
							{Key: "Token",
								Type:  "string",
								Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
							},
						},
					},
				},
			},
		},
		{
			Id:               "ws-002",
			Title:            "Тестовое окружение",
			ConnectionConfig: "postgresql://test-db:5432/test",
			CollectionItems: []entity.CollectionItem{
				{
					Id:    uuid.NewString(),
					Title: "API Тесты",
				},
			},
		},
		{
			Id:               "ws-003",
			Title:            "Продакшен сервер",
			ConnectionConfig: "postgresql://prod-db:5432/production",
			CollectionItems: []entity.CollectionItem{
				{
					Id:    uuid.NewString(),
					Title: "Мониторинг",
					Auth: entity.Auth{
						Type: "apiKey",
					},
					Event: entity.Event{},
				},
				{
					Id:    "col-005",
					Title: "Аналитика",
					Auth: entity.Auth{
						Type: "oauth2",
					},
					Event: entity.Event{},
				},
			},
		},
	}
}
