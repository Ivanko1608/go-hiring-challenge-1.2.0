package models

type Category struct {
	// internal id
	ID uint `gorm:"primaryKey"`
	// human readable identifier
	Code string `gorm:"uniqueIndex;not null"`
	Name string `gorm:"not null"`
}

func (c *Category) TableName() string {
	return "categories"
}
