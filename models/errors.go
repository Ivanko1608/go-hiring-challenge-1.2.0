package models

import "errors"

// Repositories translate gorm errors into these so handlers can map them to
// HTTP statuses without depending on gorm.
var (
	ErrProductNotFound    = errors.New("product not found")
	ErrCategoryCodeExists = errors.New("category code already exists")
)
