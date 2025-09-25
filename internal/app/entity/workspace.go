package entity

type CollectionItem struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Auth  Auth   `json:"auth"`
	Event Event  `json:"event"`
}

type Workspace struct {
	Id               string           `json:"id"`
	Title            string           `json:"title"`
	ConnectionConfig string           `json:"connectionConfig"`
	CollectionItems  []CollectionItem `json:"collectionItems"`
}

type WorkspaceListItem struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}
