package service

import "github.com/Wladim1r/auth/internal/api/repository"

type service struct {
	ur repository.UserRepository
	tr repository.TokenRepository
}

func NewServices(
	uRepo repository.UserRepository,
	tRepo repository.TokenRepository,
) (UserService, TokenService) {
	s := service{ur: uRepo, tr: tRepo}
	return &s, &s
}
