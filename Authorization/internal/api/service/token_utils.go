package service

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/Wladim1r/auth/lib/getenv"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

func createAccessJWT(userID int) (string, error) {
	ttl := getenv.GetTime("ACCESS_TTL", 30*time.Second)
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(ttl).Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := jwtToken.SignedString(
		[]byte(getenv.GetString("SECRET_KEY", "default_secret_key")),
	)
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt: %w", err)
	}

	return signedToken, nil
}

func createRefreshToken() ([]byte, error) {
	key := make([]byte, 32)
	rand.Read(key)
	hashedToken, err := bcrypt.GenerateFromPassword(key, bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to generate hash: %w", err)
	}
	return hashedToken, nil
}
