package constants

type NatsMethod string

const (
	NatsMethodPublish   NatsMethod = "publish"
	NatsMethodRequest   NatsMethod = "request"
	NatsMethodSubscribe NatsMethod = "subscribe"
)

func NormalizeNatsMethod(m NatsMethod) NatsMethod {
	switch m {
	case NatsMethodRequest, NatsMethodSubscribe:
		return m
	default:
		return NatsMethodPublish
	}
}
