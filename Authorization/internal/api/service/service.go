// Package service
package service

import (
	"github.com/Wladim1r/auth/internal/api/repository"
	"github.com/Wladim1r/auth/internal/models"
)

type Service interface {
	CreateUser(name string, password []byte) error
	DeleteUser(name string) error
	SelectPwdByName(name string) (string, error)
	CheckUserExists(name string) error
}

type service struct {
	r repository.UsersDB
}

func NewService(repo repository.UsersDB) Service {
	return &service{
		r: repo,
	}
}

func (s *service) CreateUser(name string, password []byte) error {
	user := models.User{
		Name:     name,
		Password: string(password),
	}

	return s.r.CreateUser(&user)

}

func (s *service) DeleteUser(name string) error {
	return s.r.DeleteUser(name)
}

func (s *service) SelectPwdByName(name string) (string, error) {
	return s.r.SelectPwdByName(name)
}

func (s *service) CheckUserExists(name string) error {
	return s.r.CheckUserExists(name)
}
