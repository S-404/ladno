package constants

import "strings"

type KafkaMethod string

const (
	KafkaMethodProduce KafkaMethod = "produce"
	KafkaMethodConsume KafkaMethod = "consume"
)

func NormalizeKafkaMethod(m KafkaMethod) KafkaMethod {
	switch m {
	case KafkaMethodConsume:
		return m
	default:
		return KafkaMethodProduce
	}
}

const (
	KafkaSASLNone     = ""
	KafkaSASLPlain    = "PLAIN"
	KafkaSASLSCRAM256 = "SCRAM-SHA-256"
	KafkaSASLSCRAM512 = "SCRAM-SHA-512"
)

func NormalizeKafkaSASL(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case KafkaSASLPlain:
		return KafkaSASLPlain
	case KafkaSASLSCRAM256:
		return KafkaSASLSCRAM256
	case KafkaSASLSCRAM512:
		return KafkaSASLSCRAM512
	default:
		return KafkaSASLNone
	}
}
