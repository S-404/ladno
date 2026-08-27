package store

import (
	"errors"
	"fmt"
	"net"
	"net/http"
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

type wsKey struct {
	collectionID string
	itemID       string
}

type WsMessage struct {
	Dir  string // "out" | "in"
	Text string
	Time time.Time
}

type wsSession struct {
	conn    *websocket.Conn
	url     string
	writeMu sync.Mutex
}

type wsDialer interface {
	Dial(rawURL string, extra http.Header, cb func(*websocket.Conn, error))
}

type wsEnvVars interface {
	ActiveVariables() map[string]string
}

type wsLog interface {
	Append(entry *entity.LogEntry)
}

type wsWorkspace interface {
	GetItem() binding.Untyped
	GetSelectedWorkspace() *entity.Workspace
}

type wsSettings interface {
	GetMessageLimit() int
}

type WsStore struct {
	mu               sync.Mutex
	sessions         map[wsKey]*wsSession
	gens             map[wsKey]uint64
	messages         map[wsKey][]WsMessage
	listeners        []func()
	messageListeners []func()
	wsService        wsDialer
	envStore         wsEnvVars
	logStore         wsLog
	workspace        wsWorkspace
	settings         wsSettings
	activeWorkspace  string
}

func NewWsStore(
	svc wsDialer,
	envStore wsEnvVars,
	logStore wsLog,
	workspace wsWorkspace,
	settings wsSettings,
) *WsStore {
	s := &WsStore{
		sessions:  map[wsKey]*wsSession{},
		gens:      map[wsKey]uint64{},
		messages:  map[wsKey][]WsMessage{},
		wsService: svc,
		envStore:  envStore,
		logStore:  logStore,
		workspace: workspace,
		settings:  settings,
	}
	workspace.GetItem().AddListener(binding.NewDataListener(s.onWorkspaceChanged))
	return s
}

func (s *WsStore) onWorkspaceChanged() {
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

func (s *WsStore) messageLimit() int {
	if s.settings == nil {
		return 1000
	}
	return s.settings.GetMessageLimit()
}

func (s *WsStore) Connect(collectionID, itemID string, req entity.WsRequest, onDone func(ok bool, status string)) {
	if collectionID == "" || itemID == "" {
		if onDone != nil {
			onDone(false, "missing request id")
		}
		return
	}
	key := wsKey{collectionID: collectionID, itemID: itemID}

	s.mu.Lock()
	s.gens[key]++
	gen := s.gens[key]
	old := s.sessions[key]
	delete(s.sessions, key)
	s.mu.Unlock()
	if old != nil {
		closeWSConn(old)
	}

	vars := map[string]string{}
	if s.envStore != nil {
		vars = s.envStore.ActiveVariables()
	}
	req = applyWsEnv(req, vars)
	resolved, err := service.ResolveWSURL(req.URL)
	if err != nil {
		s.logConnect(req.URL, false, err.Error())
		s.finishConnect(onDone, false, "Failed: "+err.Error())
		return
	}
	hdr := service.ExtraWSHeaders(resolved, req.Headers)
	s.wsService.Dial(resolved, hdr, func(conn *websocket.Conn, dialErr error) {
		fyne.Do(func() {
			if dialErr != nil {
				s.logConnect(resolved, false, dialErr.Error())
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
			sess := &wsSession{conn: conn, url: resolved}
			s.sessions[key] = sess
			s.mu.Unlock()
			go s.readLoop(key, sess)
			s.notifyConnectionChange()
			s.logConnect(resolved, true, "")
			s.finishConnect(onDone, true, "Connected to "+resolved)
		})
	})
}

func (s *WsStore) finishConnect(onDone func(ok bool, status string), ok bool, status string) {
	if onDone != nil {
		onDone(ok, status)
	}
}

func (s *WsStore) readLoop(key wsKey, sess *wsSession) {
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
	sess.conn.SetReadLimit(1 << 20)
	for {
		_, data, err := sess.conn.ReadMessage()
		if err != nil {
			if !isQuietWSClose(err) {
				s.logWSError(sess.url, err.Error())
			}
			return
		}
		text := string(data)
		s.appendMessage(key, WsMessage{Dir: "in", Text: text, Time: time.Now()})
		s.logInbound(sess.url, text)
	}
}

func (s *WsStore) Send(collectionID, itemID, message string, onDone func(err error)) {
	message = s.resolveEnvString(message)
	s.mu.Lock()
	sess := s.sessions[wsKey{collectionID: collectionID, itemID: itemID}]
	s.mu.Unlock()
	if sess == nil {
		if onDone != nil {
			onDone(fmt.Errorf("not connected"))
		}
		return
	}
	sess.writeMu.Lock()
	err := sess.conn.WriteMessage(websocket.TextMessage, []byte(message))
	sess.writeMu.Unlock()
	if err != nil {
		s.logWSError(sess.url, err.Error())
		if onDone != nil {
			onDone(err)
		}
		return
	}
	s.appendMessage(wsKey{collectionID: collectionID, itemID: itemID}, WsMessage{
		Dir:  "out",
		Text: message,
		Time: time.Now(),
	})
	s.logOutbound(sess.url, message)
	if onDone != nil {
		onDone(nil)
	}
}

func (s *WsStore) IsConnected(collectionID, itemID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[wsKey{collectionID: collectionID, itemID: itemID}]
	return ok
}

func (s *WsStore) ConnectedItemKeys() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]bool, len(s.sessions))
	for key := range s.sessions {
		out[key.collectionID+"/"+key.itemID] = true
	}
	return out
}

func (s *WsStore) AddConnectionListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.listeners = append(s.listeners, fn)
	s.mu.Unlock()
}

func (s *WsStore) notifyConnectionChange() {
	s.mu.Lock()
	fns := append([]func(){}, s.listeners...)
	s.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (s *WsStore) AddMessageListener(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.messageListeners = append(s.messageListeners, fn)
	s.mu.Unlock()
}

func (s *WsStore) notifyMessageChange() {
	s.mu.Lock()
	fns := append([]func(){}, s.messageListeners...)
	s.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

func (s *WsStore) appendMessage(key wsKey, msg WsMessage) {
	limit := s.messageLimit()
	s.mu.Lock()
	s.messages[key] = trimKeepNewest(append(s.messages[key], msg), limit)
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *WsStore) MessagesText(collectionID, itemID string, all bool) string {
	s.mu.Lock()
	list := append([]WsMessage(nil), s.messages[wsKey{collectionID: collectionID, itemID: itemID}]...)
	s.mu.Unlock()
	if len(list) == 0 {
		return ""
	}
	if !all {
		list = list[len(list)-1:]
	}
	parts := make([]string, 0, len(list))
	for _, m := range list {
		parts = append(parts, formatWsMessage(m))
	}
	return strings.Join(parts, "\n")
}

func (s *WsStore) ClearMessages(collectionID, itemID string) {
	s.mu.Lock()
	delete(s.messages, wsKey{collectionID: collectionID, itemID: itemID})
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *WsStore) TrimMessagesToLimit() {
	limit := s.messageLimit()
	s.mu.Lock()
	for key, list := range s.messages {
		s.messages[key] = trimKeepNewest(list, limit)
	}
	s.mu.Unlock()
	s.notifyMessageChange()
}

func (s *WsStore) Disconnect(collectionID, itemID string) {
	key := wsKey{collectionID: collectionID, itemID: itemID}
	s.mu.Lock()
	s.gens[key]++
	sess := s.sessions[key]
	delete(s.sessions, key)
	s.mu.Unlock()
	if sess != nil {
		closeWSConn(sess)
	}
	s.notifyConnectionChange()
}

func (s *WsStore) DisconnectCollection(collectionID string) {
	s.mu.Lock()
	var toClose []*wsSession
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
		closeWSConn(sess)
	}
	if len(toClose) > 0 {
		s.notifyConnectionChange()
	}
}

func (s *WsStore) DisconnectAll() {
	s.mu.Lock()
	toClose := make([]*wsSession, 0, len(s.sessions))
	for key, sess := range s.sessions {
		s.gens[key]++
		toClose = append(toClose, sess)
		delete(s.sessions, key)
	}
	s.mu.Unlock()
	for _, sess := range toClose {
		closeWSConn(sess)
	}
	if len(toClose) > 0 {
		s.notifyConnectionChange()
	}
}

func (s *WsStore) logConnect(url string, ok bool, errMsg string) {
	if s.logStore == nil {
		return
	}
	detail := fmt.Sprintf("── WS CONNECT ──\nURL: %s\n", url)
	msg := fmt.Sprintf("WS connect %s", url)
	if ok {
		msg = fmt.Sprintf("OK WS connect %s", url)
		detail += "Status: connected\n"
	} else {
		msg = fmt.Sprintf("ERR WS connect %s: %s", url, errMsg)
		detail += "Status: failed\nError: " + errMsg + "\n"
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "ws",
		Message: msg,
		Detail:  detail,
		IsError: !ok,
	})
}

func (s *WsStore) logOutbound(url, data string) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "ws",
		Message: fmt.Sprintf("WS out %s", url),
		Detail:  fmt.Sprintf("── WS OUT ──\nURL: %s\nPayload:\n%s\n", url, data),
	})
}

func (s *WsStore) logInbound(url, data string) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:      "ws",
		Message:   fmt.Sprintf("WS in %s", url),
		Detail:    fmt.Sprintf("── WS IN ──\nURL: %s\nPayload:\n%s\n", url, data),
		Highlight: true,
	})
}

func (s *WsStore) logWSError(url, errMsg string) {
	if s.logStore == nil {
		return
	}
	s.logStore.Append(&entity.LogEntry{
		Kind:    "ws",
		Message: fmt.Sprintf("WS error %s: %s", url, errMsg),
		Detail:  fmt.Sprintf("── WS ERROR ──\nURL: %s\nError: %s\n", url, errMsg),
		IsError: true,
	})
}

func (s *WsStore) resolveEnvString(input string) string {
	if s.envStore == nil || input == "" {
		return input
	}
	return utils.SubstituteEnvVars(input, s.envStore.ActiveVariables())
}

func closeWSConn(sess *wsSession) {
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

func isQuietWSClose(err error) bool {
	if err == nil {
		return true
	}
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
		return true
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "close 1000") ||
		strings.Contains(msg, "close 1001")
}

func formatWsMessage(m WsMessage) string {
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

func applyWsEnv(req entity.WsRequest, vars map[string]string) entity.WsRequest {
	if len(vars) == 0 {
		return req
	}
	req.URL = utils.SubstituteEnvVars(req.URL, vars)
	req.Message = utils.SubstituteEnvVars(req.Message, vars)
	if len(req.Headers) > 0 {
		out := make([]entity.Variable, len(req.Headers))
		for i, h := range req.Headers {
			out[i] = entity.Variable{
				Key:   utils.SubstituteEnvVars(h.Key, vars),
				Value: utils.SubstituteEnvVars(h.Value, vars),
				Type:  h.Type,
			}
		}
		req.Headers = out
	}
	return req
}
