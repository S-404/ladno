package entity

type Address struct {
	City   string `json:"city"`
	Street string `json:"street"`
}

type Person struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Website string `json:"website"`
	Address Address
}
