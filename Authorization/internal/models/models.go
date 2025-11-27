// Package models
package models

import "time"

type User struct {
	ID            uint           `gorm:"primaryKey"`
	Name          string         `gorm:"unique;not null"`
	Password      string         `gorm:"not null"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID"`
}

type RefreshToken struct {
	ID         uint      `gorm:"primaryKey"`
	HashToken  string    `gorm:"unique;not null"`
	ExspiresAt time.Time `gorm:"not null"`

	UserID uint
	User   User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type UserRequest struct {
	Name     string `json:"name"     binding:"required"`
	Password string `json:"password" binding:"required"`
}
