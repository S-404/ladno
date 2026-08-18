package entity

import "time"

type RestBodyMode string

const (
	RestBodyRaw      RestBodyMode = "raw"
	RestBodyFormData RestBodyMode = "form-data"
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
	// Auth — уже resolved (без Inherited); применяется после env-подстановки.
	Auth Auth
	// SecretHeaderKeys — заголовки, добавленные auth (для редaktion в preview/логах).
	SecretHeaderKeys []string
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
	Request    *RestRequestSnapshot
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
}
