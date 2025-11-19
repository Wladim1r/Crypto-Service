// Package hashpwd
package hashpwd

import (
	repo "github.com/Wladim1r/auth/internal/api/repository"
	"golang.org/x/crypto/bcrypt"
)

func HashPwd(db repo.UsersDB, pwd []byte, name string) ([]byte, error) {
	hashPas, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return hashPas, nil
}
