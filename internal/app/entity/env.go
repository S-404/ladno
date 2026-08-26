package entity

// EnvVariable — одна переменная окружения (key/value).
type EnvVariable struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Enabled  bool   `json:"enabled"`
	IsSecret bool   `json:"isSecret,omitempty"`
}

// Env — именованный набор переменных для подстановки {{key}} в REST-запросах.
type Env struct {
	Id        string        `json:"id"`
	Name      string        `json:"name"`
	Variables []EnvVariable `json:"variables"`
}

// EnvLightWeight — элемент списка без полного набора переменных.
type EnvLightWeight struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
