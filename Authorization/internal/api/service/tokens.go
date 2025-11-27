package service

import (
	"fmt"
	"time"

	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
)

type TokenService interface {
	SaveToken(userID int, expAt time.Time) (access, refresh string, err error)
	RefreshAccessToken(
		refreshToken string,
		userID int,
		expAt time.Time,
	) (access, refresh string, err error)
	DeleteToken(shashedToken string) error
}

func (s *service) SaveToken(userID int, expAt time.Time) (string, string, error) {

	user, err := s.ur.SelectUserByID(uint(userID))
	if err != nil {
		return "", "", err
	}

	access, err := createAccessJWT(userID)
	if err != nil {
		return "", "", err
	}

	refresh, err := createRefreshToken()
	if err != nil {
		return "", "", err
	}

	refreshToken := models.RefreshToken{
		HashToken:  string(refresh),
		ExspiresAt: expAt,
		UserID:     user.ID,
	}

	if err := s.tr.SaveToken(refreshToken); err != nil {
		return "", "", err
	}

	return access, string(refresh), err
}

func (s *service) RefreshAccessToken(
	refreshToken string,
	userID int,
	expAt time.Time,
) (string, string, error) {
	refToken, err := s.tr.FindTokenByHash(refreshToken)
	if err != nil {
		return "", "", err
	}

	if time.Now().Unix() > refToken.ExspiresAt.Unix() {
		return "", "", fmt.Errorf("%w: life time finished", errs.ErrTokenTTL)
	}

	access, err := createAccessJWT(userID)
	if err != nil {
		return "", "", err
	}

	refresh, err := createRefreshToken()
	if err != nil {
		return "", "", err
	}

	newRefToken := models.RefreshToken{
		HashToken:  string(refresh),
		ExspiresAt: expAt,
		UserID:     uint(userID),
	}

	if err := s.tr.UpdateToken(refreshToken, newRefToken); err != nil {
		return "", "", err
	}

	return access, string(refresh), nil
}

func (s *service) DeleteToken(hashedToken string) error {
	return s.tr.DeleteToken(hashedToken)
}
