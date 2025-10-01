package entity

type CollectionItem struct {
	Id      string           `json:"id"`
	Name    string           `json:"name"`
	Request *ItemRequest     `json:"request"`
	Item    []CollectionItem `json:"item"`
}

type Collection struct {
	Id    string           `json:"id"`
	Name  string           `json:"name"`
	Auth  Auth             `json:"auth"`
	Event Event            `json:"event"`
	Item  []CollectionItem `json:"item"`
}

type Workspace struct {
	Id               string       `json:"id"`
	Name             string       `json:"name"`
	ConnectionConfig string       `json:"connectionConfig"`
	Collection       []Collection `json:"collection"`
}

type WorkspaceLightWeight struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
