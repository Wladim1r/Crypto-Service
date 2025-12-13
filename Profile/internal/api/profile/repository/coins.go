package repository

import (
	"errors"
	"fmt"

	"github.com/Wladim1r/profile/internal/models"
	"github.com/Wladim1r/profile/lib/errs"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type CoinsRepository interface {
	GetCoins(userID uint) ([]*models.Coin, error)
	AddCoin(coin *models.Coin) error
	UpdateCoin(userID uint, symbol string, quantity decimal.Decimal) error
	DeleteCoin(userID uint, symbol string) error
}

func (pr *repository) GetCoins(userID uint) ([]*models.Coin, error) {
	var coins []*models.Coin

	if err := pr.db.Where("user_id = ?", userID).Find(&coins).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", errs.ErrRecordingWNF, err.Error())
		}
		return nil, fmt.Errorf("%w: %s", errs.ErrDB, err.Error())

	}

	return coins, nil
}

func (pr *repository) AddCoin(coin *models.Coin) error {
	if err := pr.db.Create(&coin).Error; err != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, err.Error())
	}

	return nil
}

func (pr *repository) UpdateCoin(userID uint, symbol string, quantity decimal.Decimal) error {
	if err := pr.db.Model(&models.Coin{}).Where("user_id = ? AND symbol = ?", userID, symbol).Update("quantity", quantity).Error; err != nil {
		return fmt.Errorf("%w: %s", errs.ErrDB, err.Error())
	}

	return nil
}

func (pr *repository) DeleteCoin(userID uint, symbol string) error {
	if err := pr.db.Where("user_id = ? AND symbol = ?", userID, symbol).Delete(&models.Coin{}).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: %s", errs.ErrRecordingWNF, err.Error())
		}
		return fmt.Errorf("%w: %s", errs.ErrDB, err.Error())
	}

	return nil
}
