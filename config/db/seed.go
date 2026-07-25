package db

import (
	"zelko/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var seededDefaultUserAuthID uint64

func SeedDatabase() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&models.VisibilityOptions).Error; err != nil {
			return err
		}

		return nil
	})
}
