package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/utils"
	"github.com/segmentio/kafka-go"
)

type kafkaSubKey struct {
	collectionID string
	itemID       string
}

type kafkaActiveConsume struct {
	cancel context.CancelFunc
	topic  string
}

type KafkaMessage struct {
	Topic     string
	Key       string
	Value     string
	Partition int
	Offset    int64
	Time      time.Time
}

type kafkaConnState struct {
	brokers []string
	display string
}

// kafkaService is the broker surface KafkaStore needs.
type kafkaService interface {
	Dial(conn entity.KafkaConnection, cb func(brokers []string, res service.KafkaConnectResult))
	Produce(brokers []string, topic, key string, headers []kafka.Header, payload []byte) error
	Consume(ctx context.Context, brokers []string, topic, groupID string, handler func(service.KafkaInbound)) error
}

// kafkaEnvVars is the env lookup KafkaStore needs.
type kafkaEnvVars interface {
	ActiveVariables() map[string]string
}

// kafkaLog is the log append surface KafkaStore needs.
type kafkaLog interface {
	Append(entry *entity.LogEntry)
}

// kafkaWorkspace is the workspace surface KafkaStore watches.
type kafkaWorkspace interface {
	GetItem() binding.Untyped
	GetSelectedWorkspace() *entity.Workspace
}

// kafkaSettings is the settings surface KafkaStore reads.
type kafkaSettings interface {
	GetMessageLimit() int
}

type KafkaStore struct {
	mu               sync.Mutex
	conns            map[string]*kafkaConnState
	subs             map[kafkaSubKey]*kafkaActiveConsume
	messages         map[string][]KafkaMessage
	listeners        []func()
	messageListeners []func()
	kafkaService     kafkaService
	envStore         kafkaEnvVars
	logStore         kafkaLog
	workspace        kafkaWorkspace
	settings         kafkaSettings
	activeWorkspace  string
}

func NewKafkaStore(
	svc kafkaService,
	envStore kafkaEnvVars,
	logStore kafkaLog,
	workspace kafkaWorkspace,
	settings kafkaSettings,
) *KafkaStore {
	s := &KafkaStore{
		conns:        map[string]*kafkaConnState{},
		subs:         map[kafkaSubKey]*kafkaActiveConsume{},
		messages:     map[string][]KafkaMessage{},
		kafkaService: svc,
		envStore:     envStore,
		logStore:     logStore,
		workspace:    workspace,
		settings:     settings,
	}
	workspace.GetItem().AddListener(binding.NewDataListener(s.onWorkspaceChanged))
	return s
}

func (s *KafkaStore) onWorkspaceChanged() {
	ws := s.workspace.GetSelectedWorkspace()
	id := ""
	if ws != nil {
		id = ws.Id
	}
	if id == s.activeWorkspace {
		return
	}
	s.activeWorkspace = id
	s.DisconnectAll()
}

func (s *KafkaStore) messageLimit() int {
	if s.settings == nil {
		return 1000
	}
	return s.settings.GetMessageLimit()
}

func (s *KafkaStore) Connect(collectionID, collectionName string, conn entity.KafkaConnection, onDone func(ok bool, status string)) {
	if collectionID == "" {
		if onDone != nil {
			onDone(false, "missing collection id")
		}
		return
	}
	conn = applyKafkaEnv(conn, s.envStore.ActiveVariables())
	s.kafkaService.Dial(conn, func(brokers []string, res service.KafkaConnectResult) {
		fyne.Do(func() {
			if res.Error != "" || len(brokers) == 0 {
				s.logConnect(collectionName, res, false)
				if onDone != nil {
					onDone(false, res.Error)
				}
				return
			}
			s.mu.Lock()
			s.clearCollectionLocked(collectionID)
			s.conns[collectionID] = &kafkaConnState{
				brokers: brokers,
				display: res.Brokers,
			}
			s.mu.Unlock()
			s.notifyConnectionChange()
			s.logConnect(collectionName, res, true)
			if onDone != nil {
				onDone(true, fmt.Sprintf("Connected to %s", res.Brokers))
			}
		})
	})
}

func (s *KafkaStore) IsConnected(collectionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.conns[collectionID]
	return ok
}

func (s *KafkaStore) ConnectedIDs() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.conns))
	for id := range s.conns {
		out[id] = true
	}
	return out
}

func (s *KafkaStore) AddConnectionListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.listeners = append(s.listeners, fn)
	s.mu.Unlock()
}

func (s *KafkaStore) notifyConnectionChange() {
	s.mu.Lock()
	fns := append([]func(){}, s.listeners...)
	s.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (s *KafkaStore) AddMessageListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.messageListeners = append(s.messageListeners, fn)
	s.mu.Unlock()
}

func (s *KafkaStore) notifyMessageChange() {
	s.mu.Lock()
	fns := append([]func(){}, s.messageListeners...)
	s.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (s *KafkaStore) AppendMessage(collectionID string, msg KafkaMessage) {
	if collectionID == "" {
		return
	}
	limit := s.messageLimit()
	s.mu.Lock()
	list := append(s.messages[collectionID], msg)
	s.messages[collectionID] = trimKeepNewest(list, limit)
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *KafkaStore) MessagesText(collectionID, topic, filter string, showAll bool) string {
	s.mu.Lock()
	list := append([]KafkaMessage(nil), s.messages[collectionID]...)
	s.mu.Unlock()

	topic = strings.TrimSpace(topic)
	filter = strings.ToLower(strings.TrimSpace(filter))
	matched := make([]KafkaMessage, 0, len(list))
	for _, m := range list {
		if topic != "" && m.Topic != topic {
			continue
		}
		if filter != "" && !kafkaMessageMatchesFilter(m, filter) {
			continue
		}
		matched = append(matched, m)
	}
	if !showAll && len(matched) > 0 {
		matched = matched[len(matched)-1:]
	}

	var b strings.Builder
	for i, m := range matched {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("[%s] %s p=%d off=%d\n",
			m.Time.Format("15:04:05.000"), m.Topic, m.Partition, m.Offset))
		if m.Key != "" {
			b.WriteString("key: ")
			b.WriteString(m.Key)
			b.WriteByte('\n')
		}
		b.WriteString(m.Value)
	}
	return b.String()
}

func kafkaMessageMatchesFilter(m KafkaMessage, filter string) bool {
	if strings.Contains(strings.ToLower(m.Topic), filter) {
		return true
	}
	if strings.Contains(strings.ToLower(m.Key), filter) {
		return true
	}
	if strings.Contains(strings.ToLower(m.Value), filter) {
		return true
	}
	return false
}

func (s *KafkaStore) ClearMessages(collectionID, topic string) {
	s.mu.Lock()
	topic = strings.TrimSpace(topic)
	list := s.messages[collectionID]
	if topic == "" {
		delete(s.messages, collectionID)
	} else {
		kept := make([]KafkaMessage, 0, len(list))
		for _, m := range list {
			if m.Topic != topic {
				kept = append(kept, m)
			}
		}
		if len(kept) == 0 {
			delete(s.messages, collectionID)
		} else {
			s.messages[collectionID] = kept
		}
	}
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *KafkaStore) TrimMessagesToLimit() {
	limit := s.messageLimit()
	s.mu.Lock()
	for id, list := range s.messages {
		s.messages[id] = trimKeepNewest(list, limit)
	}
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *KafkaStore) Disconnect(collectionID string) {
	s.mu.Lock()
	s.clearCollectionLocked(collectionID)
	s.mu.Unlock()
	s.notifyConnectionChange()
}

func (s *KafkaStore) DisconnectAll() {
	s.mu.Lock()
	ids := map[string]struct{}{}
	for id := range s.conns {
		ids[id] = struct{}{}
	}
	for key := range s.subs {
		ids[key.collectionID] = struct{}{}
	}
	for id := range ids {
		s.clearCollectionLocked(id)
	}
	s.mu.Unlock()
	s.notifyConnectionChange()
}

func (s *KafkaStore) clearCollectionLocked(collectionID string) {
	for key, active := range s.subs {
		if key.collectionID != collectionID {
			continue
		}
		if active != nil && active.cancel != nil {
			active.cancel()
		}
		delete(s.subs, key)
	}
	delete(s.conns, collectionID)
}

func (s *KafkaStore) IsConsuming(collectionID, itemID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	active, ok := s.subs[kafkaSubKey{collectionID: collectionID, itemID: itemID}]
	return ok && active != nil && active.cancel != nil
}

func (s *KafkaStore) StopConsume(collectionID, itemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := kafkaSubKey{collectionID: collectionID, itemID: itemID}
	if active, ok := s.subs[key]; ok && active != nil && active.cancel != nil {
		active.cancel()
	}
	delete(s.subs, key)
}

func (s *KafkaStore) Run(collectionID, itemID string, method constants.KafkaMethod, req entity.KafkaRequest, onDone func(err error)) {
	req = applyKafkaRequestEnv(req, s.envStore.ActiveVariables())
	method = constants.NormalizeKafkaMethod(method)

	s.mu.Lock()
	st := s.conns[collectionID]
	connected := st != nil && len(st.brokers) > 0
	var brokers []string
	if connected {
		brokers = append([]string{}, st.brokers...)
	}
	s.mu.Unlock()

	if connected {
		s.runConnected(collectionID, itemID, method, req, brokers, onDone)
		return
	}

	colName, conn, ok := s.lookupCollectionKafka(collectionID)
	if !ok {
		err := fmt.Errorf("not connected and no Kafka collection settings")
		s.logKafkaError(req.Topic, err.Error())
		if onDone != nil {
			onDone(err)
		}
		return
	}
	s.Connect(collectionID, colName, conn, func(ok bool, status string) {
		if !ok {
			err := fmt.Errorf("%s", status)
			if onDone != nil {
				onDone(err)
			}
			return
		}
		s.mu.Lock()
		st := s.conns[collectionID]
		var brokers []string
		if st != nil {
			brokers = append([]string{}, st.brokers...)
		}
		s.mu.Unlock()
		if len(brokers) == 0 {
			err := fmt.Errorf("connect succeeded but brokers missing")
			s.logKafkaError(req.Topic, err.Error())
			if onDone != nil {
				onDone(err)
			}
			return
		}
		s.runConnected(collectionID, itemID, method, req, brokers, onDone)
	})
}

func (s *KafkaStore) lookupCollectionKafka(collectionID string) (name string, conn entity.KafkaConnection, ok bool) {
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return "", entity.KafkaConnection{}, false
	}
	for _, col := range ws.Collections {
		if col.Id != collectionID {
			continue
		}
		if col.Kafka == nil {
			return col.Name, entity.KafkaConnection{}, false
		}
		return col.Name, *col.Kafka, true
	}
	return "", entity.KafkaConnection{}, false
}

func (s *KafkaStore) runConnected(
	collectionID, itemID string,
	method constants.KafkaMethod,
	req entity.KafkaRequest,
	brokers []string,
	onDone func(err error),
) {
	headers := variablesToKafkaHeaders(req.Headers)
	switch method {
	case constants.KafkaMethodConsume:
		s.runConsume(collectionID, itemID, brokers, req.Topic, onDone)
	default:
		go func() {
			err := s.kafkaService.Produce(brokers, req.Topic, req.Key, headers, []byte(req.Payload))
			fyne.Do(func() {
				if err != nil {
					s.logKafkaError(req.Topic, err.Error())
					if onDone != nil {
						onDone(err)
					}
					return
				}
				s.AppendMessage(collectionID, KafkaMessage{
					Topic: req.Topic,
					Key:   req.Key,
					Value: req.Payload,
					Time:  time.Now(),
				})
				s.logKafkaProduce(req.Topic, req.Key, req.Payload)
				if onDone != nil {
					onDone(nil)
				}
			})
		}()
	}
}

func (s *KafkaStore) runConsume(collectionID, itemID string, brokers []string, topic string, onDone func(err error)) {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		err := fmt.Errorf("topic required")
		s.logKafkaError(topic, err.Error())
		if onDone != nil {
			onDone(err)
		}
		return
	}

	key := kafkaSubKey{collectionID: collectionID, itemID: itemID}
	s.mu.Lock()
	if prev, ok := s.subs[key]; ok && prev != nil && prev.cancel != nil {
		prev.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.subs[key] = &kafkaActiveConsume{cancel: cancel, topic: topic}
	s.mu.Unlock()

	groupID := fmt.Sprintf("ladno-%s-%s", collectionID, itemID)
	go func() {
		err := s.kafkaService.Consume(ctx, brokers, topic, groupID, func(in service.KafkaInbound) {
			s.AppendMessage(collectionID, KafkaMessage{
				Topic:     in.Topic,
				Key:       in.Key,
				Value:     in.Value,
				Partition: in.Partition,
				Offset:    in.Offset,
				Time:      in.Time,
			})
			s.logKafkaInbound(in)
		})
		fyne.Do(func() {
			s.mu.Lock()
			if cur, ok := s.subs[key]; ok && cur != nil && cur.cancel != nil {
				// Only clear if this consume session is still current.
				select {
				case <-ctx.Done():
					delete(s.subs, key)
				default:
					if err != nil {
						delete(s.subs, key)
					}
				}
			}
			s.mu.Unlock()
			if err != nil && ctx.Err() == nil {
				s.logKafkaError(topic, err.Error())
			}
		})
	}()

	if onDone != nil {
		onDone(nil)
	}
}

func (s *KafkaStore) logConnect(collectionName string, res service.KafkaConnectResult, ok bool) {
	if s.logStore == nil {
		return
	}
	name := collectionName
	if name == "" {
		name = "kafka"
	}
	detail := fmt.Sprintf("── KAFKA CONNECT ──\nCollection: %s\nBrokers: %s\nDuration: %d ms\n",
		name, res.Brokers, res.Duration.Milliseconds())
	msg := fmt.Sprintf("Kafka connect %s", res.Brokers)
	isErr := !ok
	if ok {
		msg = fmt.Sprintf("OK Kafka connect %s (%d ms)", res.Brokers, res.Duration.Milliseconds())
		detail += "Status: connected\n"
	} else {
		msg = fmt.Sprintf("ERR Kafka connect %s: %s", res.Brokers, res.Error)
		detail += "Status: failed\nError: " + res.Error + "\n"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "kafka",
		Message: msg,
		Detail:  detail,
		IsError: isErr,
	})
}

func (s *KafkaStore) logKafkaProduce(topic, key, payload string) {
	if s.logStore == nil {
		return
	}
	detail := fmt.Sprintf("── KAFKA OUT ──\nTopic: %s\nKey: %s\nPayload:\n%s\n", topic, key, payload)
	s.logStore.Append(&entity.LogEntry{
		Kind:    "kafka",
		Message: fmt.Sprintf("Kafka produce %s", topic),
		Detail:  detail,
	})
}

func (s *KafkaStore) logKafkaInbound(in service.KafkaInbound) {
	if s.logStore == nil {
		return
	}
	var b strings.Builder
	b.WriteString("── KAFKA IN ──\n")
	b.WriteString(fmt.Sprintf("Topic: %s\nPartition: %d\nOffset: %d\n", in.Topic, in.Partition, in.Offset))
	if in.Key != "" {
		b.WriteString("Key: ")
		b.WriteString(in.Key)
		b.WriteByte('\n')
	}
	if len(in.Headers) > 0 {
		b.WriteString("Headers:\n")
		for _, h := range in.Headers {
			b.WriteString(fmt.Sprintf("  %s: %s\n", h.Key, string(h.Value)))
		}
	}
	b.WriteString("Value:\n")
	b.WriteString(in.Value)
	b.WriteByte('\n')
	s.logStore.Append(&entity.LogEntry{
		Kind:      "kafka",
		Message:   fmt.Sprintf("Kafka in %s", in.Topic),
		Detail:    b.String(),
		Highlight: true,
	})
}

func (s *KafkaStore) logKafkaError(topic, errMsg string) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "kafka",
		Message: fmt.Sprintf("Kafka error %s: %s", topic, errMsg),
		Detail:  fmt.Sprintf("── KAFKA ERROR ──\nTopic: %s\nError: %s\n", topic, errMsg),
		IsError: true,
	})
}

func applyKafkaEnv(conn entity.KafkaConnection, vars map[string]string) entity.KafkaConnection {
	conn.Brokers = utils.SubstituteEnvVars(conn.Brokers, vars)
	return conn
}

func applyKafkaRequestEnv(req entity.KafkaRequest, vars map[string]string) entity.KafkaRequest {
	req.Topic = utils.SubstituteEnvVars(req.Topic, vars)
	req.Key = utils.SubstituteEnvVars(req.Key, vars)
	req.Payload = utils.SubstituteEnvVars(req.Payload, vars)
	if len(req.Headers) > 0 {
		out := make([]entity.Variable, len(req.Headers))
		copy(out, req.Headers)
		for i := range out {
			out[i].Key = utils.SubstituteEnvVars(out[i].Key, vars)
			out[i].Value = utils.SubstituteEnvVars(out[i].Value, vars)
		}
		req.Headers = out
	}
	return req
}

func variablesToKafkaHeaders(vars []entity.Variable) []kafka.Header {
	if len(vars) == 0 {
		return nil
	}
	out := make([]kafka.Header, 0, len(vars))
	for _, v := range vars {
		if v.Key == "" {
			continue
		}
		out = append(out, kafka.Header{Key: v.Key, Value: []byte(v.Value)})
	}
	return out
}
