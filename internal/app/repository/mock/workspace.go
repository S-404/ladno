package mock

import (
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

// WorkspaceData возвращает мок данные для workspace
func WorkspaceData() []*entity.Workspace {
	return []*entity.Workspace{
		{
			Id:               "ws-001",
			Name:             "Основное рабочее пространство",
			ConnectionConfig: "postgresql://localhost:5432/main",
			Collections: []entity.Collection{
				{
					Id:   "c-001",
					Name: "Коллекция1",
					Items: []entity.CollectionItem{
						{
							Id:   "i-001",
							Name: "Пользователи",
							Request: &entity.ItemRequest{
								Method: constants.PUT,
								Url: entity.RequestUrl{
									Raw: "{{host}}/api/user/:id",
									Variable: []entity.Variable{
										{Key: "id", Value: "1", Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Id:               "ws-002",
			Name:             "Заказы",
			ConnectionConfig: "postgresql://localhost:5432/main",
			Collections: []entity.Collection{
				{
					Id:   "c-002",
					Name: "Коллекция1",
					Items: []entity.CollectionItem{
						{
							Id:   "i-002",
							Name: "Order",
							Item: []entity.CollectionItem{
								{
									Id:   "i-003",
									Name: "create",
									Request: &entity.ItemRequest{
										Method: constants.POST,
										Url: entity.RequestUrl{
											Raw: "{{host}}/api/order/",
										},
									},
								},
								{
									Id:   "i-004",
									Name: "upd",
									Request: &entity.ItemRequest{
										Method: constants.PUT,
										Url: entity.RequestUrl{
											Raw: "{{host}}/api/order/:id",
											Variable: []entity.Variable{
												{Key: "id", Value: "22", Type: "string"},
											},
										},
									},
								},
							},
						},
						{
							Id:   "i-005",
							Name: "another Order",
							Item: []entity.CollectionItem{
								{
									Id:   "i-006",
									Name: "get",
									Request: &entity.ItemRequest{
										Method: constants.GET,
										Url: entity.RequestUrl{
											Raw: "{{host}}/api/another/order/",
										},
									},
								},
								{
									Id:   "i-007",
									Name: "upd",
									Request: &entity.ItemRequest{
										Method: constants.PUT,
										Url: entity.RequestUrl{
											Raw: "{{host}}/api/another/order/:id",
											Variable: []entity.Variable{
												{Key: "id", Value: "33", Type: "string"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
