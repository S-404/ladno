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
			Name:             "Demo",
			ConnectionConfig: "",
			Collections: []entity.Collection{
				{
					Id:   "c-001",
					Name: "JSONPlaceholder",
					Items: []entity.CollectionItem{
						{
							Id:   "i-001",
							Name: "Get post",
							Request: &entity.ItemRequest{
								Method: constants.GET,
								Url: entity.RequestUrl{
									Raw: "https://jsonplaceholder.typicode.com/posts/:id",
									Variable: []entity.Variable{
										{Key: "id", Value: "1", Type: "string"},
									},
								},
							},
						},
						{
							Id:   "i-002",
							Name: "List posts",
							Request: &entity.ItemRequest{
								Method: constants.GET,
								Url: entity.RequestUrl{
									Raw: "https://jsonplaceholder.typicode.com/posts?_limit=5",
								},
							},
						},
						{
							Id:   "i-003",
							Name: "Create post",
							Request: &entity.ItemRequest{
								Method: constants.POST,
								Header: []entity.Variable{
									{Key: "Content-Type", Value: "application/json"},
								},
								Url: entity.RequestUrl{
									Raw: "https://jsonplaceholder.typicode.com/posts",
								},
							},
						},
						{
							Id:   "i-004",
							Name: "Update post",
							Request: &entity.ItemRequest{
								Method: constants.PUT,
								Header: []entity.Variable{
									{Key: "Content-Type", Value: "application/json"},
								},
								Url: entity.RequestUrl{
									Raw: "https://jsonplaceholder.typicode.com/posts/:id",
									Variable: []entity.Variable{
										{Key: "id", Value: "1", Type: "string"},
									},
								},
							},
						},
						{
							Id:   "i-005",
							Name: "Delete post",
							Request: &entity.ItemRequest{
								Method: constants.DELETE,
								Url: entity.RequestUrl{
									Raw: "https://jsonplaceholder.typicode.com/posts/:id",
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
	}
}
