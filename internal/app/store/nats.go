package store

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"github.com/nats-io/nats.go"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/utils"
)

type INatsStore interface {
	Connect(collectionID, collectionName string, conn entity.NatsConnection, onDone func(ok bool, status string))
	IsConnected(collectionID string) bool
	Disconnect(collectionID string)
}

type NatsStore struct {
	mu          sync.Mutex
	conns       map[string]*nats.Conn
	natsService service.INatsService
	envStore    IEnvStore
	logStore    ILogStore
}

func NewNatsStore(natsService service.INatsService, envStore IEnvStore, logStore ILogStore) *NatsStore {
	return &NatsStore{
		conns:       map[string]*nats.Conn{},
		natsService: natsService,
		envStore:    envStore,
		logStore:    logStore,
	}
}

func (s *NatsStore) Connect(collectionID, collectionName string, conn entity.NatsConnection, onDone func(ok bool, status string)) {
	if collectionID == "" {
		if onDone != nil {
			onDone(false, "missing collection id")
		}
		return
	}

	conn = applyNatsEnv(conn, s.envStore)

	s.natsService.Connect(conn, func(nc *nats.Conn, res service.NatsConnectResult) {
		fyne.Do(func() {
			if res.Error != "" {
				s.logConnect(collectionName, res, false)
				if onDone != nil {
					onDone(false, "Failed: "+res.Error)
				}
				return
			}

			s.mu.Lock()
			if prev, ok := s.conns[collectionID]; ok && prev != nil && prev.IsConnected() {
				prev.Close()
			}
			s.conns[collectionID] = nc
			s.mu.Unlock()

			s.logConnect(collectionName, res, true)
			if onDone != nil {
				status := fmt.Sprintf("Connected to %s", res.URL)
				if res.ServerID != "" {
					status = fmt.Sprintf("Connected to %s (server %s)", res.URL, res.ServerID)
				}
				onDone(true, status)
			}
		})
	})
}

func (s *NatsStore) IsConnected(collectionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	nc := s.conns[collectionID]
	return nc != nil && nc.IsConnected()
}

func (s *NatsStore) Disconnect(collectionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nc, ok := s.conns[collectionID]; ok && nc != nil {
		nc.Close()
		delete(s.conns, collectionID)
	}
}

func (s *NatsStore) logConnect(collectionName string, res service.NatsConnectResult, ok bool) {
	if s.logStore == nil {
		return
	}
	name := collectionName
	if name == "" {
		name = "nats"
	}
	msg := fmt.Sprintf("NATS connect %s (%d ms)", res.URL, res.Duration.Milliseconds())
	detail := fmt.Sprintf("── NATS CONNECT ──\nCollection: %s\nURL: %s\nDuration: %d ms\n",
		name, res.URL, res.Duration.Milliseconds())
	if ok {
		msg = fmt.Sprintf("OK NATS connect %s (%d ms)", res.URL, res.Duration.Milliseconds())
		if res.ServerID != "" {
			detail += "Server ID: " + res.ServerID + "\n"
		}
		detail += "Status: connected\n"
	} else {
		msg = fmt.Sprintf("ERR NATS connect %s (%d ms): %s", res.URL, res.Duration.Milliseconds(), res.Error)
		detail += "Status: failed\nError: " + res.Error + "\n"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "nats",
		Message: msg,
		Detail:  detail,
		IsError: !ok,
	})
}

func applyNatsEnv(conn entity.NatsConnection, envStore IEnvStore) entity.NatsConnection {
	if envStore == nil {
		return conn
	}
	vars := envStore.ActiveVariables()
	if len(vars) == 0 {
		return conn
	}
	conn.Host = utils.SubstituteEnvVars(conn.Host, vars)
	conn.Port = utils.SubstituteEnvVars(conn.Port, vars)
	conn.Token = utils.SubstituteEnvVars(conn.Token, vars)
	return conn
}
