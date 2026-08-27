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
					Type: constants.CollectionTypeHTTP,
					Auth: entity.Auth{Type: constants.AuthTypeNoAuth},
					Items: []entity.CollectionItem{
						{
							Id:   "i-001",
							Name: "Get post",
							Request: &entity.ItemRequest{
								Method: constants.GET,
								Url: entity.RequestUrl{
									Raw: "{{baseUrl}}/posts/:id",
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
									Raw: "{{baseUrl}}/posts?_limit=5",
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
									{Key: "Authorization", Value: "Bearer {{token}}"},
								},
								Url: entity.RequestUrl{
									Raw: "{{baseUrl}}/posts",
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
									Raw: "{{baseUrl}}/posts/:id",
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
									Raw: "{{baseUrl}}/posts/:id",
									Variable: []entity.Variable{
										{Key: "id", Value: "1", Type: "string"},
									},
								},
							},
						},
						{
							Id:   "f-001",
							Name: "Extras",
							Auth: entity.Auth{Type: constants.AuthTypeInherited},
							Item: []entity.CollectionItem{
								{
									Id:   "i-006",
									Name: "Health",
									Request: &entity.ItemRequest{
										Method: constants.GET,
										Url: entity.RequestUrl{
											Raw: "{{baseUrl}}/",
										},
									},
								},
							},
						},
					},
				},
				{
					Id:   "c-002",
					Name: "Demo gRPC",
					Type: constants.CollectionTypeHTTP,
					Auth: entity.Auth{Type: constants.AuthTypeNoAuth},
					Items: []entity.CollectionItem{
						{
							Id:   "g-001",
							Name: "GetUser",
							Request: &entity.ItemRequest{
								Grpc: &entity.GrpcRequest{
									Target: "localhost:50051",
									Method: "demo.UserService/GetUser",
									Metadata: []entity.Variable{
										{Key: "authorization", Value: "Bearer {{token}}"},
									},
									Message: `{"id": "1"}`,
								},
							},
						},
					},
				},
				{
					Id:   "c-003",
					Name: "Demo WebSocket",
					Type: constants.CollectionTypeHTTP,
					Auth: entity.Auth{Type: constants.AuthTypeNoAuth},
					Items: []entity.CollectionItem{
						{
							Id:   "w-001",
							Name: "Echo",
							Request: &entity.ItemRequest{
								Ws: &entity.WsRequest{
									URL:     "wss://echo.websocket.events",
									Message: "hello",
								},
							},
						},
						{
							Id:   "sio-001",
							Name: "Chat",
							Request: &entity.ItemRequest{
								SocketIO: &entity.SocketIORequest{
									URL:     "http://localhost:3000",
									Event:   "message",
									Payload: `{"text":"hello"}`,
								},
							},
						},
					},
				},
				{
					Id:   "c-004",
					Name: "Demo NATS",
					Type: constants.CollectionTypeNATS,
					Nats: &entity.NatsConnection{
						Host:  "{{natsHost}}",
						Port:  "{{natsPort}}",
						Token: "{{natsToken}}",
					},
					Items: []entity.CollectionItem{
						{
							Id:   "n-001",
							Name: "demo.events",
							Request: &entity.ItemRequest{
								Nats: &entity.NatsRequest{
									Subject: "{{natsSubject}}",
									Headers: []entity.Variable{
										{Key: "X-Token", Value: "{{token}}"},
									},
									Payload: `{"ok": true, "token": "{{token}}"}`,
								},
							},
						},
					},
				},
				{
					Id:   "c-005",
					Name: "Demo Kafka",
					Type: constants.CollectionTypeKafka,
					Kafka: &entity.KafkaConnection{
						Brokers: "{{kafkaBrokers}}",
					},
					Items: []entity.CollectionItem{
						{
							Id:   "k-001",
							Name: "demo.events",
							Request: &entity.ItemRequest{
								Kafka: &entity.KafkaRequest{
									Topic: "{{kafkaTopic}}",
									Key:   "",
									Headers: []entity.Variable{
										{Key: "X-Token", Value: "{{token}}"},
									},
									Payload: `{"ok": true, "token": "{{token}}"}`,
								},
							},
						},
					},
				},
			},
		},
	}
}
