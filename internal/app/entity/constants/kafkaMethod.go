package constants

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
