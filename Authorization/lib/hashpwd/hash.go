// Package hashpwd
package hashpwd

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPwd(pwd []byte) ([]byte, error) {
	hashPas, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return hashPas, nil
}
