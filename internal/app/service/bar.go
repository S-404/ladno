package service

import (
	"encoding/json"
	"fmt"
	"github.com/s-404/goose/internal/app/entity"
	"io"
	"net/http"
	"time"
)

type IBarService interface {
	DoSomething()
	FetchPersons() ([]entity.Person, error)
	FetchPersonsAsync(cb func([]entity.Person, error))
}

type BarService struct {
	name string
}

func NewBarService() *BarService {
	return &BarService{
		name: "foo repo",
	}
}

func (s *BarService) DoSomething() {
	fmt.Printf("'%s' doing smth", s.name)
}

func (s *BarService) FetchPersons() ([]entity.Person, error) {
	// Искусственная задержка на 2 секунды
	time.Sleep(3 * time.Second)
	url := "https://jsonplaceholder.typicode.com/users"
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var users []entity.Person
	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *BarService) FetchPersonsAsync(cb func([]entity.Person, error)) {
	go func() {
		// Искусственная задержка на 2 секунды
		time.Sleep(3 * time.Second)

		url := "https://jsonplaceholder.typicode.com/users"
		resp, err := http.Get(url)
		if err != nil {
			cb(nil, err)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			cb(nil, err)
			return
		}

		var users []entity.Person
		err = json.Unmarshal(body, &users)
		if err != nil {
			cb(nil, err)
			return
		}

		cb(users, nil)
	}()
}
