package repository

import (
	"fmt"

	"github.com/Wladim1r/auth/internal/models"
	"github.com/Wladim1r/auth/lib/errs"
)

type TokenRepository interface {
	SaveToken(token models.RefreshToken) error
	FindTokenByHash(hashedToken string) (*models.RefreshToken, error)
	UpdateToken(hashtoken string, refreshToken models.RefreshToken) error
	DeleteToken(hashedToken string) error
}

func (r *repository) SaveToken(token models.RefreshToken) error {
	result := r.db.Create(&token)

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWNC
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}

func (r *repository) FindTokenByHash(hashedToken string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken

	result := r.db.Where("hash_token = ?", hashedToken).First(&refreshToken)

	if result.RowsAffected == 0 {
		return nil, errs.ErrRecordingWNF
	}

	if result.Error != nil {
		return nil, fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return &refreshToken, nil
}

func (r *repository) UpdateToken(hashtoken string, refreshToken models.RefreshToken) error {
	updates := map[string]interface{}{
		"hash_token":  refreshToken.HashToken,
		"exspires_at": refreshToken.ExspiresAt,
	}

	if err := r.db.Model(&models.RefreshToken{}).
		Where("hash_token = ?", hashtoken).Updates(updates).Error; err != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, err.Error())
	}

	return nil
}

func (r *repository) DeleteToken(hashedToken string) error {

	result := r.db.Where("hash_token = ?", hashedToken).Delete(&models.RefreshToken{})

	if result.RowsAffected == 0 {
		return errs.ErrRecordingWNC
	}

	if result.Error != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, result.Error.Error())
	}

	return nil
}
