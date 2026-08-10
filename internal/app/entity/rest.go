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
}

// RestResponse — результат выполнения HTTP-запроса.
type RestResponse struct {
	StatusCode int
	Status     string
	Headers    map[string][]string
	Body       string
	Duration   time.Duration
	Error      string
}

// RestDraft — черновик запроса в UI.
type RestDraft struct {
	Method     string
	URL        string
	PathParams map[string]string
	Headers    []Variable
	BodyMode   RestBodyMode
	RawBody    string
	FormData   []Variable
}
