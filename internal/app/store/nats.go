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

type natsTrafficEvent struct {
	collectionID string
	subject      string
	data         string
	header       nats.Header
	matched      bool
}

type natsOutboundNote struct {
	collectionID string
	subject      string
	via          string
}

// natsService is the broker surface NatsStore needs.
type natsService interface {
	Connect(conn entity.NatsConnection, cb func(*nats.Conn, service.NatsConnectResult))
	Publish(nc *nats.Conn, subject string, headers nats.Header, payload []byte) error
	Request(nc *nats.Conn, subject string, headers nats.Header, payload []byte, timeout time.Duration) (*nats.Msg, error)
	Subscribe(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error)
}

// natsEnvVars is the env lookup/mutation NatsStore needs.
type natsEnvVars interface {
	ActiveVariables() map[string]string
	UpsertActiveVar(key, value string) bool
	ClearActiveVar(key string) bool
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

	trafficMu             sync.Mutex
	trafficPending        []natsTrafficEvent
	trafficFlushScheduled bool
	outboundPending       []natsOutboundNote
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
		hdr := cloneNatsHeader(msg.Header)
		matched := s.hasMatchingSub(collectionID, subj)
		s.enqueueTraffic(natsTrafficEvent{
			collectionID: collectionID,
			subject:      subj,
			data:         data,
			header:       hdr,
			matched:      matched,
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

func cloneNatsHeader(h nats.Header) nats.Header {
	if len(h) == 0 {
		return nil
	}
	out := make(nats.Header, len(h))
	for k, vals := range h {
		cp := make([]string, len(vals))
		copy(cp, vals)
		out[k] = cp
	}
	return out
}

func (s *NatsStore) enqueueTraffic(ev natsTrafficEvent) {
	s.trafficMu.Lock()
	s.trafficPending = append(s.trafficPending, ev)
	limit := s.messageLimit()
	if maxPending := limit * 2; maxPending > 0 && len(s.trafficPending) > maxPending {
		s.trafficPending = s.trafficPending[len(s.trafficPending)-maxPending:]
	}
	schedule := !s.trafficFlushScheduled
	if schedule {
		s.trafficFlushScheduled = true
	}
	s.trafficMu.Unlock()
	if schedule {
		fyne.Do(s.flushTraffic)
	}
}

func (s *NatsStore) flushTraffic() {
	s.trafficMu.Lock()
	pending := s.trafficPending
	s.trafficPending = nil
	s.trafficFlushScheduled = false
	s.trafficMu.Unlock()
	if len(pending) == 0 {
		return
	}

	matchedAny := false
	for _, ev := range pending {
		if via, ok := s.takeOutbound(ev.collectionID, ev.subject); ok {
			s.logNatsOutbound(ev.subject, ev.data, ev.header, via)
			continue
		}
		if ev.matched {
			s.appendMessageLocked(ev.collectionID, ev.subject, ev.data)
			s.logNatsInbound(ev.subject, ev.data, ev.header, "subscribe", 0)
			matchedAny = true
			continue
		}
		s.logNatsSubject(ev.subject)
	}
	if matchedAny {
		s.notifyMessageChange()
	}
}

func (s *NatsStore) noteOutbound(collectionID, subject, via string) {
	if collectionID == "" || subject == "" {
		return
	}
	if via == "" {
		via = "publish"
	}
	s.trafficMu.Lock()
	s.outboundPending = append(s.outboundPending, natsOutboundNote{
		collectionID: collectionID,
		subject:      subject,
		via:          via,
	})
	if len(s.outboundPending) > 64 {
		s.outboundPending = s.outboundPending[len(s.outboundPending)-64:]
	}
	s.trafficMu.Unlock()
}

func (s *NatsStore) takeOutbound(collectionID, subject string) (via string, ok bool) {
	s.trafficMu.Lock()
	defer s.trafficMu.Unlock()
	for i, note := range s.outboundPending {
		if note.collectionID == collectionID && note.subject == subject {
			via = note.via
			s.outboundPending = append(s.outboundPending[:i], s.outboundPending[i+1:]...)
			return via, true
		}
	}
	return "", false
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

// SubscribedItemKeys returns "collectionID/itemID" for active subscriptions.
func (s *NatsStore) SubscribedItemKeys() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.subs))
	for key, active := range s.subs {
		if active != nil && active.sub != nil && active.sub.IsValid() {
			out[key.collectionID+"/"+key.itemID] = true
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
	s.appendMessageLocked(collectionID, subject, data)
	s.notifyMessageChange()
}

func (s *NatsStore) appendMessageLocked(collectionID, subject, data string) {
	if collectionID == "" {
		return
	}
	limit := s.messageLimit()
	s.mu.Lock()
	list := append(s.messages[collectionID], NatsMessage{
		Subject: subject,
		Data:    data,
		Time:    time.Now(),
	})
	s.messages[collectionID] = trimKeepNewest(list, limit)
	s.mu.Unlock()
}

func (s *NatsStore) Messages(collectionID, subjectPattern string) []StreamMessage {
	subjectPattern = s.resolveEnvString(subjectPattern)
	s.mu.Lock()
	list := append([]NatsMessage(nil), s.messages[collectionID]...)
	s.mu.Unlock()
	out := make([]StreamMessage, 0, len(list))
	for _, m := range list {
		if subjectPattern != "" && !natsSubjectMatch(subjectPattern, m.Subject) {
			continue
		}
		out = append(out, StreamMessage{
			Time: m.Time,
			Dir:  "in",
			Body: formatNatsBody(m),
		})
	}
	return out
}

func formatNatsBody(m NatsMessage) string {
	return utils.PrettyBody(m.Data, "")
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
		s.messages[id] = trimKeepNewest(list, limit)
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
	key := natsSubKey{collectionID: collectionID, itemID: itemID}
	if active, ok := s.subs[key]; ok && active != nil && active.sub != nil {
		_ = active.sub.Unsubscribe()
		delete(s.subs, key)
	}
	s.mu.Unlock()
	s.notifyConnectionChange()
}

func (s *NatsStore) Run(collectionID, itemID string, method constants.NatsMethod, req entity.NatsRequest, event entity.Event, onDone func(err error, scriptErr string)) {
	method = constants.NormalizeNatsMethod(method)

	var scriptErr error
	if method == constants.NatsMethodRequest && len(event.PreRequest) > 0 {
		if err := ApplyPreRequest(event.PreRequest, s.envStore); err != nil {
			scriptErr = err
		}
	}

	req = applyNatsRequestEnv(req, s.envStore.ActiveVariables())

	s.mu.Lock()
	nc := s.conns[collectionID]
	connected := nc != nil && nc.IsConnected()
	s.mu.Unlock()

	finish := func(err error) {
		msg := ""
		if scriptErr != nil {
			msg = scriptErr.Error()
			s.logScriptError(msg)
		}
		if onDone != nil {
			onDone(err, msg)
		}
	}

	if connected {
		s.runConnected(collectionID, itemID, method, req, event, &scriptErr, nc, finish)
		return
	}

	colName, conn, ok := s.lookupCollectionNats(collectionID)
	if !ok {
		err := fmt.Errorf("not connected and no NATS collection settings")
		s.logNatsError(req.Subject, err.Error())
		finish(err)
		return
	}

	s.Connect(collectionID, colName, conn, func(ok bool, status string) {
		if !ok {
			finish(fmt.Errorf("%s", status))
			return
		}
		s.mu.Lock()
		nc := s.conns[collectionID]
		s.mu.Unlock()
		if nc == nil || !nc.IsConnected() {
			err := fmt.Errorf("connect succeeded but connection missing")
			s.logNatsError(req.Subject, err.Error())
			finish(err)
			return
		}
		s.runConnected(collectionID, itemID, method, req, event, &scriptErr, nc, finish)
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
	event entity.Event,
	scriptErr *error,
	nc *nats.Conn,
	onDone func(err error),
) {
	headers := variablesToNatsHeader(req.Headers)
	payload := []byte(req.Payload)

	applyPost := func(body string) {
		if method != constants.NatsMethodRequest || len(event.PostRequest) == 0 {
			return
		}
		if err := ApplyPostRequest(body, event.PostRequest, s.envStore); err != nil {
			if scriptErr != nil && *scriptErr == nil {
				*scriptErr = err
			} else if scriptErr != nil && *scriptErr != nil {
				*scriptErr = fmt.Errorf("%w; %v", *scriptErr, err)
			}
		}
	}

	switch method {
	case constants.NatsMethodSubscribe:
		s.runSubscribe(collectionID, itemID, nc, req.Subject, onDone)
	case constants.NatsMethodRequest:
		go func() {
			s.noteOutbound(collectionID, req.Subject, "request")
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
				s.logNatsInbound(req.Subject, string(msg.Data), msg.Header, "reply", dur)
				applyPost(string(msg.Data))
				if onDone != nil {
					onDone(nil)
				}
			})
		}()
	default: // publish — исходящее ловит monitor (">") и помечается через noteOutbound
		go func() {
			s.noteOutbound(collectionID, req.Subject, "publish")
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
	s.notifyConnectionChange()

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

// logNatsSubject — прочая шина (не наша отправка и не matched subscribe).
func (s *NatsStore) logNatsSubject(subject string) {
	if s.logStore == nil {
		return
	}
	if subject == "" {
		subject = "(empty subject)"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "nats",
		Message: "· " + subject,
		Detail:  "── NATS BUS ──\nSubject: " + subject + "\n",
	})
}

func (s *NatsStore) logNatsOutbound(subject, data string, header nats.Header, via string) {
	if s.logStore == nil {
		return
	}
	if subject == "" {
		subject = "(empty subject)"
	}
	var b strings.Builder
	b.WriteString("── NATS OUT ──\n")
	b.WriteString("Via: ")
	b.WriteString(via)
	b.WriteByte('\n')
	b.WriteString("Subject: ")
	b.WriteString(subject)
	b.WriteByte('\n')
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
	prefix := "→ "
	if via == "request" {
		prefix = "→ req "
	} else if via == "publish" {
		prefix = "→ pub "
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "nats",
		Message: prefix + subject,
		Detail:  b.String(),
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
	prefix := "← "
	if via == "subscribe" {
		prefix = "← sub "
	} else if via == "reply" {
		prefix = "← reply "
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:      "nats",
		Message:   prefix + subject,
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

func (s *NatsStore) logScriptError(errMsg string) {
	if s.logStore == nil || errMsg == "" {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "script",
		Message: "Script error: " + errMsg,
		Detail:  errMsg,
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
