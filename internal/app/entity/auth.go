package entity

import (
	"encoding/base64"
	"strings"

	"github.com/s-404/ladno/internal/app/entity/constants"
)

func AuthVar(data []Variable, key string) string {
	for _, v := range data {
		if v.Key == key {
			return v.Value
		}
	}
	return ""
}

func AuthHasVar(data []Variable, key string) bool {
	for _, v := range data {
		if v.Key == key {
			return true
		}
	}
	return false
}

func SetAuthVar(data []Variable, key, value string) []Variable {
	for i := range data {
		if data[i].Key == key {
			data[i].Value = value
			return data
		}
	}
	return append(data, Variable{Key: key, Value: value, Type: "string"})
}

// ResolveAuth returns the first non-Inherited auth in chain (request → folders → collection).
func ResolveAuth(chain []Auth) Auth {
	for _, a := range chain {
		t := constants.NormalizeAuthType(a.Type)
		if t == constants.AuthTypeInherited || t == "" {
			continue
		}
		a.Type = t
		return a
	}
	return Auth{Type: constants.AuthTypeNoAuth}
}

// ApplyAuth merges resolved auth into a request copy (headers / body form fields).
func ApplyAuth(req RestRequest, auth Auth) RestRequest {
	auth.Type = constants.NormalizeAuthType(auth.Type)
	switch auth.Type {
	case constants.AuthTypeNoAuth, constants.AuthTypeInherited, constants.AuthTypeJSON, "":
		return req
	case constants.AuthTypeBasic, constants.AuthTypeBearer:
		for _, h := range AuthGeneratedHeaders(auth) {
			req.Headers = upsertHeader(req.Headers, h.Key, h.Value)
			req.SecretHeaderKeys = appendSecretHeaderKey(req.SecretHeaderKeys, h.Key)
		}
	case constants.AuthTypeAPIKey:
		key := AuthVar(auth.Data, constants.AuthDataKey)
		val := AuthVar(auth.Data, constants.AuthDataValue)
		if key == "" {
			return req
		}
		addTo := AuthVar(auth.Data, constants.AuthDataAddTo)
		if addTo == constants.AuthAddToBody {
			req.FormData = upsertHeader(req.FormData, key, val)
			req.BodyMode = RestBodyFormData
			return req
		}
		req.Headers = upsertHeader(req.Headers, key, val)
		req.SecretHeaderKeys = appendSecretHeaderKey(req.SecretHeaderKeys, key)
	}
	return req
}

// AuthGeneratedHeaders returns headers that auth would add (not body fields).
func AuthGeneratedHeaders(auth Auth) []Variable {
	auth.Type = constants.NormalizeAuthType(auth.Type)
	switch auth.Type {
	case constants.AuthTypeBasic:
		user := AuthVar(auth.Data, constants.AuthDataUsername)
		pass := AuthVar(auth.Data, constants.AuthDataPassword)
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		return []Variable{{Key: "Authorization", Value: "Basic " + token}}
	case constants.AuthTypeBearer:
		token := AuthVar(auth.Data, constants.AuthDataToken)
		prefix := AuthVar(auth.Data, constants.AuthDataPrefix)
		if !AuthHasVar(auth.Data, constants.AuthDataPrefix) {
			prefix = constants.AuthDefaultTokenPrefix
		}
		val := token
		if strings.TrimSpace(prefix) != "" {
			val = strings.TrimSpace(prefix) + " " + token
		}
		return []Variable{{Key: "Authorization", Value: val}}
	case constants.AuthTypeAPIKey:
		key := AuthVar(auth.Data, constants.AuthDataKey)
		val := AuthVar(auth.Data, constants.AuthDataValue)
		if key == "" {
			return nil
		}
		if AuthVar(auth.Data, constants.AuthDataAddTo) == constants.AuthAddToBody {
			return nil
		}
		return []Variable{{Key: key, Value: val}}
	default:
		return nil
	}
}

// ApplyAuthHeaders merges token/API key headers from auth into the header list.
func ApplyAuthHeaders(headers []Variable, auth Auth) []Variable {
	for _, h := range AuthGeneratedHeaders(auth) {
		headers = upsertHeader(headers, h.Key, h.Value)
	}
	return headers
}

// SocketIOAuthJSON is the CONNECT-packet payload. Token/API key go in headers, not here.
func SocketIOAuthJSON(auth Auth, legacy string) string {
	t := constants.NormalizeAuthType(auth.Type)
	switch t {
	case constants.AuthTypeJSON:
		return strings.TrimSpace(AuthVar(auth.Data, constants.AuthDataJSON))
	case constants.AuthTypeNoAuth, constants.AuthTypeInherited, "":
		return strings.TrimSpace(legacy)
	default:
		return ""
	}
}

// EffectiveSocketIOAuth maps a leftover authJson field onto JSON auth when type is unset.
func EffectiveSocketIOAuth(auth Auth, legacyJSON string) Auth {
	t := constants.NormalizeAuthType(auth.Type)
	if t == constants.AuthTypeInherited || t == constants.AuthTypeNoAuth || t == "" {
		if js := strings.TrimSpace(legacyJSON); js != "" {
			return Auth{
				Type: constants.AuthTypeJSON,
				Data: []Variable{{Key: constants.AuthDataJSON, Value: js, Type: "string"}},
			}
		}
		if t == constants.AuthTypeInherited || t == "" {
			return Auth{Type: constants.AuthTypeNoAuth}
		}
	}
	return auth
}

func upsertHeader(headers []Variable, key, value string) []Variable {
	for i := range headers {
		if headers[i].Key == key {
			headers[i].Value = value
			return headers
		}
	}
	return append(headers, Variable{Key: key, Value: value})
}

func appendSecretHeaderKey(keys []string, key string) []string {
	for _, k := range keys {
		if strings.EqualFold(k, key) {
			return keys
		}
	}
	return append(keys, key)
}
