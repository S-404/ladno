package entity

import "time"

type RestBodyMode string

const (
	RestBodyRaw        RestBodyMode = "raw"
	RestBodyFormData   RestBodyMode = "form-data"
	RestBodyURLEncoded RestBodyMode = "x-www-form-urlencoded"
)

// RestRequest — снимок запроса для отправки.
type RestRequest struct {
	Method     string
	URL        string
	PathParams map[string]string
	Headers    []Variable
	BodyMode   RestBodyMode
	RawBody    string
	FormData   []Variable
	URLEncoded []Variable
	// Auth — уже resolved (без Inherited); применяется после env-подстановки.
	Auth Auth
	// SecretHeaderKeys — заголовки, добавленные auth (для редaktion в preview/логах).
	SecretHeaderKeys []string
	// PreRequest — скрипты до отправки (запись/очистка active env).
	PreRequest []PreRequestEnvEvent
	// PostRequest — скрипты после ответа (запись в active env из body).
	PostRequest []PostRequestEnvEvent
}

// RestResponse — результат выполнения HTTP-запроса.
type RestResponse struct {
	Method     string
	URL        string // итоговый URL после path params
	StatusCode int
	Status     string
	Headers    map[string][]string
	Body       string
	Duration   time.Duration
	Error      string
	// ScriptError — ошибка pre/postRequest (пусто если ок / скриптов нет).
	ScriptError string
	Request     *RestRequestSnapshot
}

// RestDraft — черновик запроса в UI.
type RestDraft struct {
	Method     string
	URL        string
	PathParams map[string]string
	Headers    []Variable
	Auth       Auth
	BodyMode   RestBodyMode
	RawBody    string
	FormData   []Variable
	URLEncoded []Variable
	Event      Event
}
