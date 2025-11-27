// Package service
package service

import (
	"github.com/Wladim1r/auth/internal/models"
)

type UserService interface {
	CreateUser(name string, password []byte) (int, error)
	DeleteUser(userID int) error
	SelectPwdByName(name string) (string, error)
	CheckUserExistsByName(name string) error
	CheckUserExistsByID(userID int) error
}

func (s *service) CreateUser(name string, password []byte) (int, error) {
	user := models.User{
		Name:     name,
		Password: string(password),
	}

	return s.ur.CreateUser(&user)
}

func (s *service) DeleteUser(userID int) error {
	return s.ur.DeleteUser(uint(userID))
}

func (s *service) SelectPwdByName(name string) (string, error) {
	return s.ur.SelectPwdByName(name)
}

func (s *service) CheckUserExistsByName(name string) error {
	return s.ur.CheckUserExistsByName(name)
}
func (s *service) CheckUserExistsByID(userID int) error {
	return s.ur.CheckUserExistsByID(uint(userID))
}
