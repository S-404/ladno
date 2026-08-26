package service

import (
	"testing"

	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
)

func TestKafkaClientRequiresBrokers(t *testing.T) {
	_, _, err := kafkaClient(entity.KafkaConnection{})
	if err == nil {
		t.Fatal("expected brokers required")
	}
}

func TestKafkaClientPlainSASL(t *testing.T) {
	brokers, dialer, err := kafkaClient(entity.KafkaConnection{
		Brokers:  "localhost:9092",
		SASL:     constants.KafkaSASLPlain,
		Username: "u",
		Password: "p",
		TLS:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(brokers) != 1 || brokers[0] != "localhost:9092" {
		t.Fatalf("brokers=%v", brokers)
	}
	if dialer == nil || dialer.TLS == nil || dialer.SASLMechanism == nil {
		t.Fatal("expected TLS and SASL")
	}
	if dialer.SASLMechanism.Name() != "PLAIN" {
		t.Fatalf("mech=%s", dialer.SASLMechanism.Name())
	}
}

func TestKafkaClientNoAuth(t *testing.T) {
	_, dialer, err := kafkaClient(entity.KafkaConnection{Brokers: "a:1, b:2"})
	if err != nil {
		t.Fatal(err)
	}
	if dialer.TLS != nil || dialer.SASLMechanism != nil {
		t.Fatal("expected plaintext")
	}
}
