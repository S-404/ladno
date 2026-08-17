package store

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/nats-io/nats.go"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/entity/constants"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/utils"
)

type natsSubKey struct {
	collectionID string
	itemID       string
}

type natsActiveSub struct {
	sub     *nats.Subscription
	pattern string
}

type NatsMessage struct {
	Subject string
	Data    string
	Time    time.Time
}

// natsService is the broker surface NatsStore needs.
type natsService interface {
	Connect(conn entity.NatsConnection, cb func(*nats.Conn, service.NatsConnectResult))
	Publish(nc *nats.Conn, subject string, headers nats.Header, payload []byte) error
	Request(nc *nats.Conn, subject string, headers nats.Header, payload []byte, timeout time.Duration) (*nats.Msg, error)
	Subscribe(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error)
}

// natsEnvVars is the env lookup NatsStore needs.
type natsEnvVars interface {
	ActiveVariables() map[string]string
}

// natsLog is the log append surface NatsStore needs.
type natsLog interface {
	Append(entry *entity.LogEntry)
}

// natsWorkspace is the workspace surface NatsStore watches.
type natsWorkspace interface {
	GetItem() binding.Untyped
	GetSelectedWorkspace() *entity.Workspace
}

// natsSettings is the settings surface NatsStore reads.
type natsSettings interface {
	GetMessageLimit() int
}

type NatsStore struct {
	mu               sync.Mutex
	conns            map[string]*nats.Conn
	subs             map[natsSubKey]*natsActiveSub
	monitors         map[string]*nats.Subscription // collectionID → ">" tap
	messages         map[string][]NatsMessage      // collectionID → traffic
	listeners        []func()
	messageListeners []func()
	natsService      natsService
	envStore         natsEnvVars
	logStore         natsLog
	workspace        natsWorkspace
	settings         natsSettings
	activeWorkspace  string
}

func NewNatsStore(
	svc natsService,
	envStore natsEnvVars,
	logStore natsLog,
	workspace natsWorkspace,
	settings natsSettings,
) *NatsStore {
	s := &NatsStore{
		conns:       map[string]*nats.Conn{},
		subs:        map[natsSubKey]*natsActiveSub{},
		monitors:    map[string]*nats.Subscription{},
		messages:    map[string][]NatsMessage{},
		natsService: svc,
		envStore:    envStore,
		logStore:    logStore,
		workspace:   workspace,
		settings:    settings,
	}
	workspace.GetItem().AddListener(binding.NewDataListener(s.onWorkspaceChanged))
	return s
}

func (s *NatsStore) onWorkspaceChanged() {
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

func (s *NatsStore) messageLimit() int {
	if s.settings == nil {
		return 1000
	}
	return s.settings.GetMessageLimit()
}

func (s *NatsStore) Connect(collectionID, collectionName string, conn entity.NatsConnection, onDone func(ok bool, status string)) {
	if collectionID == "" {
		if onDone != nil {
			onDone(false, "missing collection id")
		}
		return
	}

	conn = applyNatsEnv(conn, s.envStore.ActiveVariables())

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
			s.clearCollectionLocked(collectionID)
			s.conns[collectionID] = nc
			s.mu.Unlock()

			if err := s.startMonitor(collectionID, nc); err != nil {
				s.logNatsError(">", "monitor: "+err.Error())
			}
			s.notifyConnectionChange()

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

func (s *NatsStore) startMonitor(collectionID string, nc *nats.Conn) error {
	mon, err := s.natsService.Subscribe(nc, ">", func(msg *nats.Msg) {
		if msg == nil {
			return
		}
		// request/reply inbox — логируем в ветке Request как highlight
		if strings.HasPrefix(msg.Subject, "_INBOX.") {
			return
		}
		subj := msg.Subject
		data := string(msg.Data)
		hdr := msg.Header
		fyne.Do(func() {
			if s.hasMatchingSub(collectionID, subj) {
				s.AppendMessage(collectionID, subj, data)
				s.logNatsInbound(subj, data, hdr, "subscribe", 0)
				return
			}
			s.logNatsSubject(subj)
		})
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.monitors[collectionID] = mon
	s.mu.Unlock()
	return nil
}

func (s *NatsStore) hasMatchingSub(collectionID, subject string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, active := range s.subs {
		if key.collectionID != collectionID || active == nil {
			continue
		}
		if active.sub != nil && !active.sub.IsValid() {
			continue
		}
		if natsSubjectMatch(active.pattern, subject) {
			return true
		}
	}
	return false
}

func (s *NatsStore) IsConnected(collectionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	nc := s.conns[collectionID]
	return nc != nil && nc.IsConnected()
}

func (s *NatsStore) ConnectedIDs() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.conns))
	for id, nc := range s.conns {
		if nc != nil && nc.IsConnected() {
			out[id] = true
		}
	}
	return out
}

func (s *NatsStore) AddConnectionListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.listeners = append(s.listeners, fn)
	s.mu.Unlock()
}

func (s *NatsStore) notifyConnectionChange() {
	s.mu.Lock()
	listeners := append([]func(){}, s.listeners...)
	s.mu.Unlock()
	for _, fn := range listeners {
		fn()
	}
}

func (s *NatsStore) AddMessageListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.messageListeners = append(s.messageListeners, fn)
	s.mu.Unlock()
}

func (s *NatsStore) notifyMessageChange() {
	s.mu.Lock()
	listeners := append([]func(){}, s.messageListeners...)
	s.mu.Unlock()
	for _, fn := range listeners {
		fn()
	}
}

func (s *NatsStore) AppendMessage(collectionID, subject, data string) {
	if collectionID == "" {
		return
	}
	limit := s.messageLimit()
	s.mu.Lock()
	list := s.messages[collectionID]
	list = append(list, NatsMessage{
		Subject: subject,
		Data:    data,
		Time:    time.Now(),
	})
	if len(list) > limit {
		list = list[len(list)-limit:]
	}
	s.messages[collectionID] = list
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *NatsStore) MessagesText(collectionID, subjectPattern string, all bool) string {
	subjectPattern = s.resolveEnvString(subjectPattern)
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.messages[collectionID]
	matched := make([]NatsMessage, 0, len(list))
	for _, m := range list {
		if subjectPattern == "" || natsSubjectMatch(subjectPattern, m.Subject) {
			matched = append(matched, m)
		}
	}
	if len(matched) == 0 {
		return ""
	}
	limit := s.messageLimit()
	if len(matched) > limit {
		matched = matched[len(matched)-limit:]
	}
	format := func(m NatsMessage) string {
		ts := m.Time.Format("15:04:05.000")
		if m.Data == "" {
			return "[" + ts + "]"
		}
		return "[" + ts + "] " + m.Data
	}
	if !all {
		return format(matched[len(matched)-1])
	}
	parts := make([]string, 0, len(matched))
	for _, m := range matched {
		parts = append(parts, format(m))
	}
	return strings.Join(parts, "\n")
}

func (s *NatsStore) ClearMessages(collectionID, subjectPattern string) {
	subjectPattern = s.resolveEnvString(subjectPattern)
	s.mu.Lock()
	list := s.messages[collectionID]
	if subjectPattern == "" {
		delete(s.messages, collectionID)
	} else {
		kept := make([]NatsMessage, 0, len(list))
		for _, m := range list {
			if !natsSubjectMatch(subjectPattern, m.Subject) {
				kept = append(kept, m)
			}
		}
		s.messages[collectionID] = kept
	}
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *NatsStore) TrimMessagesToLimit() {
	limit := s.messageLimit()
	s.mu.Lock()
	for id, list := range s.messages {
		if len(list) > limit {
			s.messages[id] = list[len(list)-limit:]
		}
	}
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *NatsStore) Disconnect(collectionID string) {
	s.mu.Lock()
	s.clearCollectionLocked(collectionID)
	s.mu.Unlock()
	s.notifyConnectionChange()
}

func (s *NatsStore) DisconnectAll() {
	s.mu.Lock()
	ids := map[string]struct{}{}
	for id := range s.conns {
		ids[id] = struct{}{}
	}
	for id := range s.monitors {
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

func (s *NatsStore) clearCollectionLocked(collectionID string) {
	if mon, ok := s.monitors[collectionID]; ok && mon != nil {
		_ = mon.Unsubscribe()
		delete(s.monitors, collectionID)
	}
	for key, active := range s.subs {
		if key.collectionID == collectionID {
			if active != nil && active.sub != nil {
				_ = active.sub.Unsubscribe()
			}
			delete(s.subs, key)
		}
	}
	if prev, ok := s.conns[collectionID]; ok && prev != nil {
		prev.Close()
		delete(s.conns, collectionID)
	}
}

func (s *NatsStore) IsSubscribed(collectionID, itemID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	active, ok := s.subs[natsSubKey{collectionID: collectionID, itemID: itemID}]
	return ok && active != nil && active.sub != nil && active.sub.IsValid()
}

func (s *NatsStore) Unsubscribe(collectionID, itemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := natsSubKey{collectionID: collectionID, itemID: itemID}
	if active, ok := s.subs[key]; ok && active != nil && active.sub != nil {
		_ = active.sub.Unsubscribe()
		delete(s.subs, key)
	}
}

func (s *NatsStore) Run(collectionID, itemID string, method constants.NatsMethod, req entity.NatsRequest, onDone func(err error)) {
	req = applyNatsRequestEnv(req, s.envStore.ActiveVariables())
	method = constants.NormalizeNatsMethod(method)

	s.mu.Lock()
	nc := s.conns[collectionID]
	connected := nc != nil && nc.IsConnected()
	s.mu.Unlock()

	if connected {
		s.runConnected(collectionID, itemID, method, req, nc, onDone)
		return
	}

	colName, conn, ok := s.lookupCollectionNats(collectionID)
	if !ok {
		err := fmt.Errorf("not connected and no NATS collection settings")
		s.logNatsError(req.Subject, err.Error())
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
		nc := s.conns[collectionID]
		s.mu.Unlock()
		if nc == nil || !nc.IsConnected() {
			err := fmt.Errorf("connect succeeded but connection missing")
			s.logNatsError(req.Subject, err.Error())
			if onDone != nil {
				onDone(err)
			}
			return
		}
		s.runConnected(collectionID, itemID, method, req, nc, onDone)
	})
}

func (s *NatsStore) lookupCollectionNats(collectionID string) (name string, conn entity.NatsConnection, ok bool) {
	if s.workspace == nil {
		return "", entity.NatsConnection{}, false
	}
	ws := s.workspace.GetSelectedWorkspace()
	if ws == nil {
		return "", entity.NatsConnection{}, false
	}
	for i := range ws.Collections {
		if ws.Collections[i].Id != collectionID {
			continue
		}
		col := ws.Collections[i]
		if col.Nats == nil {
			return col.Name, entity.NatsConnection{}, false
		}
		return col.Name, *col.Nats, true
	}
	return "", entity.NatsConnection{}, false
}

func (s *NatsStore) runConnected(
	collectionID, itemID string,
	method constants.NatsMethod,
	req entity.NatsRequest,
	nc *nats.Conn,
	onDone func(err error),
) {
	headers := variablesToNatsHeader(req.Headers)
	payload := []byte(req.Payload)

	switch method {
	case constants.NatsMethodSubscribe:
		s.runSubscribe(collectionID, itemID, nc, req.Subject, onDone)
	case constants.NatsMethodRequest:
		go func() {
			start := time.Now()
			msg, err := s.natsService.Request(nc, req.Subject, headers, payload, 2*time.Second)
			dur := time.Since(start)
			fyne.Do(func() {
				if err != nil {
					s.logNatsError(req.Subject, err.Error())
					if onDone != nil {
						onDone(err)
					}
					return
				}
				s.AppendMessage(collectionID, req.Subject, string(msg.Data))
				s.logNatsInbound(msg.Subject, string(msg.Data), msg.Header, "request", dur)
				if onDone != nil {
					onDone(nil)
				}
			})
		}()
	default: // publish — факт попадания на сервер логирует monitor (">")
		go func() {
			err := s.natsService.Publish(nc, req.Subject, headers, payload)
			if err == nil {
				_ = nc.FlushTimeout(2 * time.Second)
			}
			fyne.Do(func() {
				if err != nil {
					s.logNatsError(req.Subject, err.Error())
					if onDone != nil {
						onDone(err)
					}
					return
				}
				if onDone != nil {
					onDone(nil)
				}
			})
		}()
	}
}

func (s *NatsStore) runSubscribe(collectionID, itemID string, nc *nats.Conn, subject string, onDone func(err error)) {
	key := natsSubKey{collectionID: collectionID, itemID: itemID}

	s.mu.Lock()
	if prev, ok := s.subs[key]; ok && prev != nil && prev.sub != nil {
		_ = prev.sub.Unsubscribe()
		delete(s.subs, key)
	}
	s.mu.Unlock()

	// Сообщения логирует monitor (">"): highlight если есть активная подписка.
	sub, err := s.natsService.Subscribe(nc, subject, func(msg *nats.Msg) {})
	if err != nil {
		s.logNatsError(subject, err.Error())
		if onDone != nil {
			onDone(err)
		}
		return
	}

	s.mu.Lock()
	s.subs[key] = &natsActiveSub{sub: sub, pattern: subject}
	s.mu.Unlock()

	if onDone != nil {
		onDone(nil)
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
	detail := fmt.Sprintf("── NATS CONNECT ──\nCollection: %s\nURL: %s\nDuration: %d ms\n",
		name, res.URL, res.Duration.Milliseconds())
	msg := fmt.Sprintf("NATS connect %s", res.URL)
	isErr := !ok
	if ok {
		msg = fmt.Sprintf("OK NATS connect %s (%d ms)", res.URL, res.Duration.Milliseconds())
		if res.ServerID != "" {
			detail += "Server ID: " + res.ServerID + "\n"
		}
		detail += "Status: connected\n"
	} else {
		msg = fmt.Sprintf("ERR NATS connect %s: %s", res.URL, res.Error)
		detail += "Status: failed\nError: " + res.Error + "\n"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "nats",
		Message: msg,
		Detail:  detail,
		IsError: isErr,
	})
}

// logNatsSubject — простая строка с subject (трафик без своей подписки).
func (s *NatsStore) logNatsSubject(subject string) {
	if s.logStore == nil {
		return
	}
	if subject == "" {
		subject = "(empty subject)"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "nats",
		Message: subject,
		Detail:  subject + "\n",
	})
}

func (s *NatsStore) logNatsInbound(subject, data string, header nats.Header, via string, dur time.Duration) {
	if s.logStore == nil {
		return
	}
	if subject == "" {
		subject = "(empty subject)"
	}
	var b strings.Builder
	b.WriteString("── NATS IN ──\n")
	b.WriteString("Via: ")
	b.WriteString(via)
	b.WriteByte('\n')
	b.WriteString("Subject: ")
	b.WriteString(subject)
	b.WriteByte('\n')
	if dur > 0 {
		b.WriteString(fmt.Sprintf("Duration: %d ms\n", dur.Milliseconds()))
	}
	if len(header) > 0 {
		b.WriteString("\nHeaders:\n")
		for k, vals := range header {
			for _, v := range vals {
				b.WriteString("  ")
				b.WriteString(k)
				b.WriteString(": ")
				b.WriteString(v)
				b.WriteByte('\n')
			}
		}
	}
	if data != "" {
		b.WriteString("\nPayload:\n")
		b.WriteString(data)
		b.WriteByte('\n')
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:      "nats",
		Message:   subject,
		Detail:    b.String(),
		Highlight: true,
	})
}

func (s *NatsStore) logNatsError(subject, errMsg string) {
	if s.logStore == nil {
		return
	}
	if subject == "" {
		subject = "(empty subject)"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "nats",
		Message: subject + ": " + errMsg,
		Detail:  fmt.Sprintf("── NATS ERROR ──\nSubject: %s\nError: %s\n", subject, errMsg),
		IsError: true,
	})
}

// natsSubjectMatch — упрощённый матчер NATS subject (token *, >).
func natsSubjectMatch(pattern, subject string) bool {
	if pattern == subject || pattern == ">" {
		return true
	}
	pp := strings.Split(pattern, ".")
	ss := strings.Split(subject, ".")
	for i := 0; i < len(pp); i++ {
		if i >= len(ss) {
			return false
		}
		if pp[i] == ">" {
			return true
		}
		if pp[i] == "*" {
			continue
		}
		if pp[i] != ss[i] {
			return false
		}
	}
	return len(pp) == len(ss)
}

func applyNatsEnv(conn entity.NatsConnection, vars map[string]string) entity.NatsConnection {
	if len(vars) == 0 {
		return conn
	}
	conn.Host = utils.SubstituteEnvVars(conn.Host, vars)
	conn.Port = utils.SubstituteEnvVars(conn.Port, vars)
	conn.Token = utils.SubstituteEnvVars(conn.Token, vars)
	return conn
}

func applyNatsRequestEnv(req entity.NatsRequest, vars map[string]string) entity.NatsRequest {
	if len(vars) == 0 {
		return req
	}
	req.Subject = utils.SubstituteEnvVars(req.Subject, vars)
	req.Payload = utils.SubstituteEnvVars(req.Payload, vars)
	if len(req.Headers) > 0 {
		headers := make([]entity.Variable, len(req.Headers))
		for i, h := range req.Headers {
			headers[i] = entity.Variable{
				Key:   utils.SubstituteEnvVars(h.Key, vars),
				Value: utils.SubstituteEnvVars(h.Value, vars),
				Type:  h.Type,
			}
		}
		req.Headers = headers
	}
	return req
}

func (s *NatsStore) resolveEnvString(input string) string {
	if s.envStore == nil || input == "" {
		return input
	}
	return utils.SubstituteEnvVars(input, s.envStore.ActiveVariables())
}

func variablesToNatsHeader(vars []entity.Variable) nats.Header {
	if len(vars) == 0 {
		return nil
	}
	h := nats.Header{}
	for _, v := range vars {
		if v.Key == "" {
			continue
		}
		h.Add(v.Key, v.Value)
	}
	if len(h) == 0 {
		return nil
	}
	return h
}
