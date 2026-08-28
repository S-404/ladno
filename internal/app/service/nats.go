package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/s-404/ladno/internal/app/entity"
)

type NatsConnectResult struct {
	URL      string
	Duration time.Duration
	Error    string
	ServerID string
}

type NatsMessage struct {
	Subject string
	Data    string
	Header  nats.Header
}

type NatsService struct{}

func NewNatsService() *NatsService {
	return &NatsService{}
}

func (s *NatsService) Connect(conn entity.NatsConnection, cb func(*nats.Conn, NatsConnectResult)) {
	go func() {
		start := time.Now()
		url := natsURL(conn)
		opts := []nats.Option{
			nats.Name("ladno"),
			nats.Timeout(5 * time.Second),
		}
		if token := strings.TrimSpace(conn.Token); token != "" {
			opts = append(opts, nats.Token(token))
		}

		nc, err := nats.Connect(url, opts...)
		dur := time.Since(start)
		if err != nil {
			cb(nil, NatsConnectResult{
				URL:      url,
				Duration: dur,
				Error:    err.Error(),
			})
			return
		}
		cb(nc, NatsConnectResult{
			URL:      url,
			Duration: dur,
			ServerID: nc.ConnectedServerId(),
		})
	}()
}

func (s *NatsService) Publish(nc *nats.Conn, subject string, headers nats.Header, payload []byte) error {
	if nc == nil || !nc.IsConnected() {
		return fmt.Errorf("not connected")
	}
	msg := &nats.Msg{Subject: subject, Data: payload, Header: headers}
	return nc.PublishMsg(msg)
}

func (s *NatsService) Request(nc *nats.Conn, subject string, headers nats.Header, payload []byte, timeout time.Duration) (*nats.Msg, error) {
	if nc == nil || !nc.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	msg := &nats.Msg{Subject: subject, Data: payload, Header: headers}
	return nc.RequestMsg(msg, timeout)
}

func (s *NatsService) Subscribe(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if nc == nil || !nc.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}
	return nc.Subscribe(subject, handler)
}

func natsURL(conn entity.NatsConnection) string {
	host := strings.TrimSpace(conn.Host)
	if host == "" {
		host = "localhost"
	}
	port := strings.TrimSpace(conn.Port)
	if port == "" {
		port = "4222"
	}
	if strings.Contains(host, "://") {
		return host
	}
	return fmt.Sprintf("nats://%s:%s", host, port)
}
