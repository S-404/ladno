package constants

type RequestMethod string

const (
	POST    RequestMethod = "POST"
	GET     RequestMethod = "GET"
	PUT     RequestMethod = "PUT"
	PATCH   RequestMethod = "PATCH"
	DELETE  RequestMethod = "DELETE"
	HEAD    RequestMethod = "HEAD"
	OPTIONS RequestMethod = "OPTIONS"
)
