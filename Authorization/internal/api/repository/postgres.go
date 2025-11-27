// Package repository
package repository

import (
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepositories(db *gorm.DB) (UserRepository, TokenRepository) {
	repo := repository{db: db}

	return &repo, &repo
}
