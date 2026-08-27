package store

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"github.com/gorilla/websocket"
	"github.com/s-404/ladno/internal/app/entity"
	"github.com/s-404/ladno/internal/app/service"
	"github.com/s-404/ladno/internal/app/utils"
)

type sioKey struct {
	collectionID string
	itemID       string
}

type SIOMessage struct {
	Dir  string // "out" | "in"
	Text string
	Time time.Time
}

type sioSession struct {
	conn         *websocket.Conn
	url          string
	namespace    string
	writeMu      sync.Mutex
	pingInterval time.Duration
	pingTimeout  time.Duration
	listening    map[string]bool
}

type sioDialer interface {
	Dial(rawURL string, extra http.Header, namespace, authJSON string, cb func(*websocket.Conn, service.SocketIOOpen, error))
}

type SocketIOStore struct {
	mu               sync.Mutex
	sessions         map[sioKey]*sioSession
	gens             map[sioKey]uint64
	messages         map[sioKey][]SIOMessage
	listeners        []func()
	messageListeners []func()
	sioService       sioDialer
	envStore         wsEnvVars
	logStore         wsLog
	workspace        wsWorkspace
	settings         wsSettings
	activeWorkspace  string
}

func NewSocketIOStore(
	svc sioDialer,
	envStore wsEnvVars,
	logStore wsLog,
	workspace wsWorkspace,
	settings wsSettings,
) *SocketIOStore {
	s := &SocketIOStore{
		sessions:   map[sioKey]*sioSession{},
		gens:       map[sioKey]uint64{},
		messages:   map[sioKey][]SIOMessage{},
		sioService: svc,
		envStore:   envStore,
		logStore:   logStore,
		workspace:  workspace,
		settings:   settings,
	}
	workspace.GetItem().AddListener(binding.NewDataListener(s.onWorkspaceChanged))
	return s
}

func (s *SocketIOStore) onWorkspaceChanged() {
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

func (s *SocketIOStore) messageLimit() int {
	if s.settings == nil {
		return 1000
	}
	return s.settings.GetMessageLimit()
}

func (s *SocketIOStore) Connect(collectionID, itemID string, req entity.SocketIORequest, onDone func(ok bool, status string)) {
	if collectionID == "" || itemID == "" {
		if onDone != nil {
			onDone(false, "missing request id")
		}
		return
	}
	key := sioKey{collectionID: collectionID, itemID: itemID}

	s.mu.Lock()
	s.gens[key]++
	gen := s.gens[key]
	old := s.sessions[key]
	delete(s.sessions, key)
	s.mu.Unlock()
	if old != nil {
		closeSIOConn(old)
	}

	vars := map[string]string{}
	if s.envStore != nil {
		vars = s.envStore.ActiveVariables()
	}
	req = applySocketIOEnv(req, vars)
	req.Headers = entity.ApplyAuthHeaders(req.Headers, req.Auth)
	authJSON := entity.SocketIOAuthJSON(req.Auth, req.AuthJSON)
	resolved, ns, err := service.ResolveSocketIOURL(req.URL)
	if err != nil {
		s.logConnect(req.URL, false, err.Error(), authJSON, nil)
		s.finishConnect(onDone, false, "Failed: "+err.Error())
		return
	}
	hdr := service.ExtraSocketIOHeaders(resolved, req.Headers)
	s.sioService.Dial(resolved, hdr, ns, authJSON, func(conn *websocket.Conn, open service.SocketIOOpen, dialErr error) {
		fyne.Do(func() {
			if dialErr != nil {
				s.logConnect(resolved, false, dialErr.Error(), authJSON, hdr)
				s.finishConnect(onDone, false, "Failed: "+dialErr.Error())
				return
			}
			s.mu.Lock()
			if s.gens[key] != gen {
				s.mu.Unlock()
				if conn != nil {
					_ = conn.Close()
				}
				return
			}
			sess := &sioSession{
				conn:         conn,
				url:          resolved,
				namespace:    ns,
				pingInterval: open.PingInterval,
				pingTimeout:  open.PingTimeout,
				listening:    map[string]bool{},
			}
			s.sessions[key] = sess
			s.mu.Unlock()
			go s.readLoop(key, sess)
			s.notifyConnectionChange()
			s.logConnect(resolved, true, "", authJSON, hdr)
			s.appendMessage(key, SIOMessage{
				Dir:  "in",
				Text: "connected ns=" + normalizeSIONamespace(ns),
				Time: time.Now(),
			})
			s.finishConnect(onDone, true, "Connected to "+resolved)
		})
	})
}

func (s *SocketIOStore) finishConnect(onDone func(ok bool, status string), ok bool, status string) {
	if onDone != nil {
		onDone(ok, status)
	}
}

func (s *SocketIOStore) readLoop(key sioKey, sess *sioSession) {
	defer func() {
		if sess.conn != nil {
			_ = sess.conn.Close()
		}
		s.mu.Lock()
		if s.sessions[key] == sess {
			delete(s.sessions, key)
		}
		s.mu.Unlock()
		s.notifyConnectionChange()
	}()
	limit := int64(1 << 20)
	if sess.conn != nil {
		sess.conn.SetReadLimit(limit)
	}
	for {
		if sess.pingInterval > 0 {
			_ = sess.conn.SetReadDeadline(time.Now().Add(sess.pingInterval + sess.pingTimeout))
		}
		_, data, err := sess.conn.ReadMessage()
		if err != nil {
			if !isQuietWSClose(err) {
				s.logSIOError(sess.url, err.Error())
			}
			return
		}
		pong, logLine, eventName, disconnect, interpErr := service.InterpretSocketIOFrame(string(data))
		if interpErr != nil {
			s.logSIOError(sess.url, interpErr.Error())
			s.appendMessage(key, SIOMessage{Dir: "in", Text: "error " + interpErr.Error(), Time: time.Now()})
			return
		}
		if pong != "" {
			sess.writeMu.Lock()
			_ = sess.conn.WriteMessage(websocket.TextMessage, []byte(pong))
			sess.writeMu.Unlock()
		}
		if logLine != "" {
			s.mu.Lock()
			show := showSIOInbound(sess.listening, eventName)
			s.mu.Unlock()
			if show {
				s.appendMessage(key, SIOMessage{Dir: "in", Text: logLine, Time: time.Now()})
			}
			s.logInbound(sess.url, logLine)
		}
		if disconnect {
			return
		}
	}
}

func (s *SocketIOStore) Emit(collectionID, itemID, event, payload string, namespace string, onDone func(err error)) {
	event = s.resolveEnvString(event)
	payload = s.resolveEnvString(payload)
	s.mu.Lock()
	sess := s.sessions[sioKey{collectionID: collectionID, itemID: itemID}]
	s.mu.Unlock()
	if sess == nil {
		if onDone != nil {
			onDone(fmt.Errorf("not connected"))
		}
		return
	}
	if namespace == "" {
		namespace = sess.namespace
	}
	frame, err := service.EncodeSocketIOEvent(namespace, event, payload)
	if err != nil {
		if onDone != nil {
			onDone(err)
		}
		return
	}
	sess.writeMu.Lock()
	err = sess.conn.WriteMessage(websocket.TextMessage, []byte(frame))
	sess.writeMu.Unlock()
	if err != nil {
		s.logSIOError(sess.url, err.Error())
		if onDone != nil {
			onDone(err)
		}
		return
	}
	line := "emit " + strings.TrimSpace(event)
	if strings.TrimSpace(payload) != "" {
		line += " " + strings.TrimSpace(payload)
	}
	s.appendMessage(sioKey{collectionID: collectionID, itemID: itemID}, SIOMessage{
		Dir:  "out",
		Text: line,
		Time: time.Now(),
	})
	s.logOutbound(sess.url, line)
	if onDone != nil {
		onDone(nil)
	}
}

func (s *SocketIOStore) Listen(collectionID, itemID, event string, on bool, onDone func(err error)) {
	event = strings.TrimSpace(s.resolveEnvString(event))
	if event == "" {
		if onDone != nil {
			onDone(fmt.Errorf("event name is empty"))
		}
		return
	}
	key := sioKey{collectionID: collectionID, itemID: itemID}
	s.mu.Lock()
	sess := s.sessions[key]
	if sess == nil {
		s.mu.Unlock()
		if onDone != nil {
			onDone(fmt.Errorf("not connected"))
		}
		return
	}
	if sess.listening == nil {
		sess.listening = map[string]bool{}
	}
	was := sess.listening[event]
	if on {
		sess.listening[event] = true
	} else {
		delete(sess.listening, event)
	}
	s.mu.Unlock()
	if was == on {
		if onDone != nil {
			onDone(nil)
		}
		return
	}
	line := "listen " + event
	if !on {
		line = "stop " + event
	}
	s.appendMessage(key, SIOMessage{Dir: "out", Text: line, Time: time.Now()})
	s.logOutbound(sess.url, line)
	if onDone != nil {
		onDone(nil)
	}
}

func (s *SocketIOStore) IsListening(collectionID, itemID, event string) bool {
	event = strings.TrimSpace(event)
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[sioKey{collectionID: collectionID, itemID: itemID}]
	if sess == nil || sess.listening == nil {
		return false
	}
	return sess.listening[event]
}

func (s *SocketIOStore) ListeningEvents(collectionID, itemID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.sessions[sioKey{collectionID: collectionID, itemID: itemID}]
	if sess == nil || len(sess.listening) == 0 {
		return nil
	}
	out := make([]string, 0, len(sess.listening))
	for ev := range sess.listening {
		out = append(out, ev)
	}
	sort.Strings(out)
	return out
}

func showSIOInbound(listening map[string]bool, eventName string) bool {
	if eventName == "" || len(listening) == 0 {
		return true
	}
	return listening[eventName]
}

func (s *SocketIOStore) IsConnected(collectionID, itemID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[sioKey{collectionID: collectionID, itemID: itemID}]
	return ok
}

func (s *SocketIOStore) ConnectedItemKeys() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.sessions))
	for key := range s.sessions {
		out[key.collectionID+"/"+key.itemID] = true
	}
	return out
}

func (s *SocketIOStore) AddConnectionListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.listeners = append(s.listeners, fn)
	s.mu.Unlock()
}

func (s *SocketIOStore) notifyConnectionChange() {
	s.mu.Lock()
	fns := append([]func(){}, s.listeners...)
	s.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (s *SocketIOStore) AddMessageListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.messageListeners = append(s.messageListeners, fn)
	s.mu.Unlock()
}

func (s *SocketIOStore) notifyMessageChange() {
	s.mu.Lock()
	fns := append([]func(){}, s.messageListeners...)
	s.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (s *SocketIOStore) appendMessage(key sioKey, msg SIOMessage) {
	limit := s.messageLimit()
	s.mu.Lock()
	s.messages[key] = trimKeepNewest(append(s.messages[key], msg), limit)
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *SocketIOStore) Messages(collectionID, itemID string) []StreamMessage {
	s.mu.Lock()
	list := append([]SIOMessage(nil), s.messages[sioKey{collectionID: collectionID, itemID: itemID}]...)
	s.mu.Unlock()
	out := make([]StreamMessage, len(list))
	for i, m := range list {
		out[i] = StreamMessage{
			Time: m.Time,
			Dir:  m.Dir,
			Body: utils.PrettyBody(m.Text, ""),
		}
	}
	return out
}

func (s *SocketIOStore) ClearMessages(collectionID, itemID string) {
	s.mu.Lock()
	delete(s.messages, sioKey{collectionID: collectionID, itemID: itemID})
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *SocketIOStore) TrimMessagesToLimit() {
	limit := s.messageLimit()
	s.mu.Lock()
	for key, list := range s.messages {
		s.messages[key] = trimKeepNewest(list, limit)
	}
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *SocketIOStore) Disconnect(collectionID, itemID string) {
	key := sioKey{collectionID: collectionID, itemID: itemID}
	s.mu.Lock()
	s.gens[key]++
	sess := s.sessions[key]
	delete(s.sessions, key)
	s.mu.Unlock()
	if sess != nil {
		closeSIOConn(sess)
	}
	s.notifyConnectionChange()
}

func (s *SocketIOStore) DisconnectCollection(collectionID string) {
	s.mu.Lock()
	var toClose []*sioSession
	for key, sess := range s.sessions {
		if key.collectionID != collectionID {
			continue
		}
		s.gens[key]++
		toClose = append(toClose, sess)
		delete(s.sessions, key)
	}
	s.mu.Unlock()
	for _, sess := range toClose {
		closeSIOConn(sess)
	}
	if len(toClose) > 0 {
		s.notifyConnectionChange()
	}
}

func (s *SocketIOStore) DisconnectAll() {
	s.mu.Lock()
	toClose := make([]*sioSession, 0, len(s.sessions))
	for key, sess := range s.sessions {
		s.gens[key]++
		toClose = append(toClose, sess)
		delete(s.sessions, key)
	}
	s.mu.Unlock()
	for _, sess := range toClose {
		closeSIOConn(sess)
	}
	if len(toClose) > 0 {
		s.notifyConnectionChange()
	}
}

func (s *SocketIOStore) logConnect(url string, ok bool, errMsg, authJSON string, hdr http.Header) {
	if s.logStore == nil {
		return
	}
	detail := fmt.Sprintf("── SOCKET.IO CONNECT ──\nURL: %s\n", url)
	if strings.TrimSpace(authJSON) == "" {
		detail += "Auth JSON: (none)\n"
	} else {
		detail += "Auth JSON: " + strings.TrimSpace(authJSON) + "\n"
	}
	if len(hdr) > 0 {
		detail += "Headers:\n"
		for _, k := range sortedHeaderKeys(hdr) {
			if strings.EqualFold(k, "Host") {
				continue
			}
			val := strings.Join(hdr[k], ", ")
			if strings.EqualFold(k, "Authorization") {
				val = "••••"
			}
			detail += "  " + k + ": " + val + "\n"
		}
	}
	msg := fmt.Sprintf("SIO connect %s", url)
	if ok {
		msg = fmt.Sprintf("OK SIO connect %s", url)
		detail += "Status: connected\n"
	} else {
		msg = fmt.Sprintf("ERR SIO connect %s: %s", url, errMsg)
		detail += "Status: failed\nError: " + errMsg + "\n"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "socketio",
		Message: msg,
		Detail:  detail,
		IsError: !ok,
	})
}

func sortedHeaderKeys(h http.Header) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *SocketIOStore) logOutbound(url, data string) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "socketio",
		Message: fmt.Sprintf("SIO out %s", url),
		Detail:  fmt.Sprintf("── SOCKET.IO OUT ──\nURL: %s\n%s\n", url, data),
	})
}

func (s *SocketIOStore) logInbound(url, data string) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:      "socketio",
		Message:   fmt.Sprintf("SIO in %s", url),
		Detail:    fmt.Sprintf("── SOCKET.IO IN ──\nURL: %s\n%s\n", url, data),
		Highlight: true,
	})
}

func (s *SocketIOStore) logSIOError(url, errMsg string) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "socketio",
		Message: fmt.Sprintf("SIO error %s: %s", url, errMsg),
		Detail:  fmt.Sprintf("── SOCKET.IO ERROR ──\nURL: %s\nError: %s\n", url, errMsg),
		IsError: true,
	})
}

func (s *SocketIOStore) resolveEnvString(input string) string {
	if s.envStore == nil || input == "" {
		return input
	}
	return utils.SubstituteEnvVars(input, s.envStore.ActiveVariables())
}

func closeSIOConn(sess *sioSession) {
	if sess == nil || sess.conn == nil {
		return
	}
	sess.writeMu.Lock()
	_ = sess.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)
	sess.writeMu.Unlock()
	_ = sess.conn.Close()
}

func formatSIOMessage(m SIOMessage) string {
	ts := m.Time.Format("15:04:05.000")
	arrow := "→"
	if m.Dir == "in" {
		arrow = "←"
	}
	body := utils.PrettyBody(m.Text, "")
	if strings.TrimSpace(body) == "" {
		return "[" + ts + "] " + arrow
	}
	return "[" + ts + "] " + arrow + " " + body
}

func applySocketIOEnv(req entity.SocketIORequest, vars map[string]string) entity.SocketIORequest {
	if len(vars) > 0 {
		req.URL = utils.SubstituteEnvVars(req.URL, vars)
		req.Path = utils.SubstituteEnvVars(req.Path, vars)
		req.Namespace = utils.SubstituteEnvVars(req.Namespace, vars)
		req.AuthJSON = utils.SubstituteEnvVars(req.AuthJSON, vars)
		req.Event = utils.SubstituteEnvVars(req.Event, vars)
		req.Payload = utils.SubstituteEnvVars(req.Payload, vars)
		req.Headers = substituteVarList(req.Headers, vars, true)
		req.Query = substituteVarList(req.Query, vars, true)
		req.Auth = applyEnvToAuth(req.Auth, vars)
	}
	req.URL = service.MergeLegacySocketIOURL(req.URL, req.Namespace)
	req.URL = service.MergeSocketIOQuery(req.URL, req.Query)
	return req
}

func substituteVarList(list []entity.Variable, vars map[string]string, keys bool) []entity.Variable {
	if len(list) == 0 {
		return list
	}
	out := make([]entity.Variable, len(list))
	for i, h := range list {
		key := h.Key
		if keys {
			key = utils.SubstituteEnvVars(h.Key, vars)
		}
		out[i] = entity.Variable{
			Key:   key,
			Value: utils.SubstituteEnvVars(h.Value, vars),
			Type:  h.Type,
		}
	}
	return out
}

func normalizeSIONamespace(ns string) string {
	ns = strings.TrimSpace(ns)
	if ns == "" {
		return "/"
	}
	if !strings.HasPrefix(ns, "/") {
		ns = "/" + ns
	}
	return ns
}
