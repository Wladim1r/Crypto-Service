package repository

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
)

type UserRepository interface {
	CreateTable()
	CreateUser(user *models.User) (int, error)
	DeleteUser(userID uint) error
	SelectPwdByName(name string) (string, error)
	CheckUserExistsByName(name string) error
	CheckUserExistsByID(userID uint) error
	SelectUserByID(userID uint) (*models.User, error)
}

func (r *repository) CreateTable() {
	// create tables
	if err := r.db.AutoMigrate(&models.User{}, &models.RefreshToken{}); err != nil {
		slog.Error("Could not create db table", "error", err.Error())
		os.Exit(1)
	}

	// check table exists or not
	if ok := r.db.Migrator().HasTable("users"); !ok {
		slog.Error("Table 'users' has not created idk")
		os.Exit(1)
	}
	if ok := r.db.Migrator().HasTable("refresh_tokens"); !ok {
		slog.Error("Table 'refresh_tokens' has not created idk")
		os.Exit(1)
	}
}

func (r *repository) CreateUser(user *models.User) (int, error) {
	result := r.db.Create(&user)

	if result.RowsAffected == 0 {
		return -1, errs.ErrRecordingWNC
	}

	if result.Error != nil {
		return -1, fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return int(user.ID), nil
}

func (r *repository) DeleteUser(userID uint) error {
	result := r.db.Where("id = ?", userID).Delete(&models.User{})

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWND
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}

func (r *repository) SelectPwdByName(name string) (string, error) {
	var user models.User

	result := r.db.Table("users").Select("password").Where("name = ?", name).Scan(&user)

	if result.RowsAffected == 0 {
		return "", errs.ErrRecordingWNF
	}

	if result.Error != nil {
		return "", fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return user.Password, nil
}

func (r *repository) CheckUserExistsByName(name string) error {
	var user models.User

	result := r.db.Table("users").Select("id").Where("name = ?", name).Scan(&user)

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWNF
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}

func (r *repository) CheckUserExistsByID(userID uint) error {
	var user models.User

	result := r.db.Table("users").Select("name").Where("id = ?", userID).Scan(&user)

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWNF
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}

func (r *repository) SelectUserByID(userID uint) (*models.User, error) {
	var user models.User

	result := r.db.Table("users").Where("id = ?", userID).Scan(&user)

	if result.RowsAffected == 0 {
		return nil, errs.ErrRecordingWNF
	}

	if result.Error != nil {
		return nil, fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return &user, nil
}
