// Package repository
package repository

import (
	"database/sql"
)

type UsersDB interface {
	CreateTable() error
	CreateUser(name string, password []byte) error
	DeleteUser(name string) error
	SelectPwdByName(name string) (string, error)
	SelectIDByName(name string) *sql.Row
	Close()
}

type usersDB struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) UsersDB {
	return &usersDB{
		db: db,
	}
}

func (db *usersDB) CreateTable() error {
	_, err := db.db.Exec(
		"CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, password TEXT)",
	)
	if err != nil {
		return err
	}

	return nil
}

func (db *usersDB) CreateUser(name string, password []byte) error {
	_, err := db.db.Exec("INSERT INTO users (name, password) VALUES (?, ?)", name, password)
	if err != nil {
		return err
	}

	return nil
}

func (db *usersDB) DeleteUser(name string) error {
	_, err := db.db.Exec("DELETE FROM users WHERE name = ?", name)
	if err != nil {
		return err
	}

	return nil
}

func (db *usersDB) SelectPwdByName(name string) (string, error) {
	row := db.db.QueryRow("SELECT password FROM users WHERE name = ?", name)

	var password string
	if err := row.Scan(&password); err != nil {
		return "", err
	}

	return password, nil
}

func (db *usersDB) SelectIDByName(name string) *sql.Row {
	return db.db.QueryRow("SELECT id FROM users WHERE name = ?", name)
}

func (db *usersDB) Close() {
	db.db.Close()
}
