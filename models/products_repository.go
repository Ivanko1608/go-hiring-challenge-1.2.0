package models

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ProductFilter narrows and paginates a product listing. Zero-valued fields
// (empty CategoryCode, invalid PriceLessThan) mean "don't filter by this".
type ProductFilter struct {
	Offset        int
	Limit         int
	CategoryCode  string
	PriceLessThan decimal.NullDecimal
}

type ProductsRepository struct {
	db *gorm.DB
}

func NewProductsRepository(db *gorm.DB) *ProductsRepository {
	return &ProductsRepository{
		db: db,
	}
}

func (r *ProductsRepository) GetProductsFiltered(ctx context.Context, f ProductFilter) ([]Product, int64, error) {
	q := r.filter(ctx, f).Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []Product
	err := q.Order("products.id").
		Offset(f.Offset).
		Limit(f.Limit).
		Find(&products).Error
	if err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *ProductsRepository) filter(ctx context.Context, f ProductFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&Product{}).Joins("Category")
	if f.CategoryCode != "" {
		q = q.Where(`"Category"."code" = ?`, f.CategoryCode)
	}
	if f.PriceLessThan.Valid {
		q = q.Where("products.price < ?", f.PriceLessThan.Decimal)
	}
	return q
}

func (r *ProductsRepository) GetProductByCode(ctx context.Context, code string) (Product, error) {
	var product Product
	err := r.db.WithContext(ctx).
		Joins("Category").
		Preload("Variants", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).
		Where("products.code = ?", code).
		First(&product).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Product{}, ErrProductNotFound
	}
	return product, err
}
