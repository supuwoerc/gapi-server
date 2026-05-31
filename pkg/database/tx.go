package database

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

func NewTxContext(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func TxFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if db, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return db
	}
	return fallback
}

type TransactionManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (m *TransactionManager) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(NewTxContext(ctx, tx))
	})
}
