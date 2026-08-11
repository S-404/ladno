package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/segmentio/kafka-go"
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

type IKafkaService interface {
	Dial(conn entity.KafkaConnection, cb func(brokers []string, res KafkaConnectResult))
	Produce(brokers []string, topic, key string, headers []kafka.Header, payload []byte) error
	Consume(ctx context.Context, brokers []string, topic, groupID string, handler func(KafkaInbound)) error
}

type KafkaService struct{}

func NewKafkaService() *KafkaService {
	return &KafkaService{}
}

func (s *KafkaService) Dial(conn entity.KafkaConnection, cb func(brokers []string, res KafkaConnectResult)) {
	go func() {
		start := time.Now()
		brokers := SplitKafkaBrokers(conn.Brokers)
		if len(brokers) == 0 {
			cb(nil, KafkaConnectResult{
				Brokers:  strings.TrimSpace(conn.Brokers),
				Duration: time.Since(start),
				Error:    "brokers required",
			})
			return
		}
		c, err := kafka.Dial("tcp", brokers[0])
		dur := time.Since(start)
		display := strings.Join(brokers, ", ")
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

func (s *KafkaService) Produce(brokers []string, topic, key string, headers []kafka.Header, payload []byte) error {
	if len(brokers) == 0 {
		return fmt.Errorf("not connected")
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

func (s *KafkaService) Consume(ctx context.Context, brokers []string, topic, groupID string, handler func(KafkaInbound)) error {
	if len(brokers) == 0 {
		return fmt.Errorf("not connected")
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
