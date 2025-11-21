// Package repository
package repository

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
	"gorm.io/gorm"
)

type UsersDB interface {
	CreateTable() error
	CreateUser(user *models.User) error
	DeleteUser(name string) error
	SelectPwdByName(name string) (string, error)
	CheckUserExists(name string) error
}

type usersDB struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) UsersDB {
	return &usersDB{
		db: db,
	}
}

func (db *usersDB) CreateTable() error {

	// create table
	if err := db.db.Migrator().CreateTable(&models.User{}); err != nil {
		slog.Error("Could not create db table", "error", err.Error())
		os.Exit(1)
	}

	// check table exists or not
	if ok := db.db.Migrator().HasTable(&models.User{}); !ok {
		slog.Error("Table has not created idk")
		os.Exit(1)
	}

	if ok := db.db.Migrator().HasTable("users"); !ok {
		slog.Error("Table has not created idk")
		os.Exit(1)
	}

	return nil
}

func (db *usersDB) CreateUser(user *models.User) error {
	result := db.db.Create(&user)

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWNC
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}

func (db *usersDB) DeleteUser(name string) error {
	result := db.db.Where("name = ?", name).Delete(&models.User{})

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWND
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}

func (db *usersDB) SelectPwdByName(name string) (string, error) {
	var user models.User

	result := db.db.Table("users").Select("password").Where("name = ?", name).Scan(&user)

	if result.RowsAffected == 0 {
		return "", errs.ErrRecordingWNF
	}

	if result.Error != nil {
		return "", fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return user.Password, nil
}

func (db *usersDB) CheckUserExists(name string) error {
	var user models.User

	result := db.db.Table("users").Select("id").Where("name = ?", name).Scan(&user)

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWNF
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}
