package models

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type CategoriesRepository struct {
	db *gorm.DB
}

func NewCategoriesRepository(db *gorm.DB) *CategoriesRepository {
	return &CategoriesRepository{
		db: db,
	}
}

func (r *CategoriesRepository) GetAllCategories(ctx context.Context) ([]Category, error) {
	var categories []Category
	if err := r.db.WithContext(ctx).Order("id").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoriesRepository) CreateCategory(ctx context.Context, c *Category) error {
	err := r.db.WithContext(ctx).Create(c).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrCategoryCodeExists
	}
	return err
}
