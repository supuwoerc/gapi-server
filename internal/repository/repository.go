package repository

import "gorm.io/gorm"

// Repository is the base data access struct embedding *gorm.DB.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new base Repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// DB returns the underlying gorm.DB instance.
func (r *Repository) DB() *gorm.DB {
	return r.db
}
