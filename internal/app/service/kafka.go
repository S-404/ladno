package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type KafkaConnectResult struct {
	Brokers  string
	Duration time.Duration
	Error    string
}

type KafkaInbound struct {
	Topic     string
	Key       string
	Value     string
	Headers   []kafka.Header
	Partition int
	Offset    int64
	Time      time.Time
}

type KafkaService struct{}

func NewKafkaService() *KafkaService {
	return &KafkaService{}
}

func (s *KafkaService) Dial(conn entity.KafkaConnection, cb func(brokers []string, res KafkaConnectResult)) {
	go func() {
		start := time.Now()
		brokers, dialer, err := kafkaClient(conn)
		display := strings.Join(brokers, ", ")
		if display == "" {
			display = strings.TrimSpace(conn.Brokers)
		}
		if err != nil {
			cb(nil, KafkaConnectResult{
				Brokers:  display,
				Duration: time.Since(start),
				Error:    err.Error(),
			})
			return
		}
		c, err := dialer.Dial("tcp", brokers[0])
		dur := time.Since(start)
		if err != nil {
			cb(nil, KafkaConnectResult{
				Brokers:  display,
				Duration: dur,
				Error:    err.Error(),
			})
			return
		}
		_ = c.Close()
		cb(brokers, KafkaConnectResult{
			Brokers:  display,
			Duration: dur,
		})
	}()
}

func (s *KafkaService) Produce(conn entity.KafkaConnection, topic, key string, headers []kafka.Header, payload []byte) error {
	brokers, dialer, err := kafkaClient(conn)
	if err != nil {
		return err
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("topic required")
	}
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Transport: &kafka.Transport{
			SASL: dialer.SASLMechanism,
			TLS:  dialer.TLS,
		},
	}
	defer func() { _ = w.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	msg := kafka.Message{
		Key:     []byte(key),
		Value:   payload,
		Headers: headers,
		Time:    time.Now(),
	}
	return w.WriteMessages(ctx, msg)
}

func (s *KafkaService) Consume(ctx context.Context, conn entity.KafkaConnection, topic, groupID string, handler func(KafkaInbound)) error {
	brokers, dialer, err := kafkaClient(conn)
	if err != nil {
		return err
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("topic required")
	}
	if groupID == "" {
		groupID = "ladno"
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		Dialer:      dialer,
		StartOffset: kafka.LastOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     500 * time.Millisecond,
	})
	defer func() { _ = r.Close() }()

	for {
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if handler != nil {
			handler(KafkaInbound{
				Topic:     msg.Topic,
				Key:       string(msg.Key),
				Value:     string(msg.Value),
				Headers:   msg.Headers,
				Partition: msg.Partition,
				Offset:    msg.Offset,
				Time:      msg.Time,
			})
		}
	}
}

func SplitKafkaBrokers(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func kafkaClient(conn entity.KafkaConnection) ([]string, *kafka.Dialer, error) {
	brokers := SplitKafkaBrokers(conn.Brokers)
	if len(brokers) == 0 {
		return nil, nil, fmt.Errorf("brokers required")
	}
	d := &kafka.Dialer{Timeout: 10 * time.Second}
	if conn.TLS {
		d.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	mech, err := kafkaSASL(conn)
	if err != nil {
		return nil, nil, err
	}
	d.SASLMechanism = mech
	return brokers, d, nil
}

func kafkaSASL(conn entity.KafkaConnection) (sasl.Mechanism, error) {
	mech := constants.NormalizeKafkaSASL(conn.SASL)
	user := strings.TrimSpace(conn.Username)
	pass := conn.Password
	switch mech {
	case constants.KafkaSASLNone:
		return nil, nil
	case constants.KafkaSASLPlain:
		return plain.Mechanism{Username: user, Password: pass}, nil
	case constants.KafkaSASLSCRAM256:
		return scram.Mechanism(scram.SHA256, user, pass)
	case constants.KafkaSASLSCRAM512:
		return scram.Mechanism(scram.SHA512, user, pass)
	default:
		return nil, nil
	}
}
