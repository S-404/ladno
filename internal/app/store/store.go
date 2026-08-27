package store

import (
	"github.com/s-404/ladno/internal/app/service"
)

type Store struct {
	Cookie    *CookieStore
	Draft     *DraftStore
	Env       *EnvStore
	Kafka     *KafkaStore
	Log       *LogStore
	Nats      *NatsStore
	Rest      *RestStore
	Selection *SelectionStore
	Settings  *SettingsStore
	Workspace *WorkspaceStore
	Ws        *WsStore
}

func NewStore(svc *service.Service) *Store {
	settingsStore := NewSettingsStore()
	logStore := NewLogStore(settingsStore)
	wsStore := NewWorkspaceStore(svc.Workspace)
	selStore := NewSelectionStore(wsStore)
	cookieStore := NewCookieStore()
	envStore := NewEnvStore(svc.Env, settingsStore, wsStore)
	draftStore := NewDraftStore(wsStore, selStore, envStore)
	envStore.SetDraftSync(draftStore)
	return &Store{
		Cookie:    cookieStore,
		Draft:     draftStore,
		Env:       envStore,
		Kafka:     NewKafkaStore(svc.Kafka, envStore, logStore, wsStore, settingsStore),
		Log:       logStore,
		Nats:      NewNatsStore(svc.Nats, envStore, logStore, wsStore, settingsStore),
		Rest:      NewRestStore(svc.Rest, envStore, logStore, cookieStore),
		Selection: selStore,
		Settings:  settingsStore,
		Workspace: wsStore,
		Ws:        NewWsStore(svc.Ws, envStore, logStore, wsStore, settingsStore),
	}
}
