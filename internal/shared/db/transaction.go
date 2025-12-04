// Package db provides database utilities including transaction management.
package db

import (
	"context"

	"gorm.io/gorm"
)

// txKey is the context key for storing transaction.
type txKey struct{}

// TransactionManager manages database transactions.
type TransactionManager struct {
	db *gorm.DB
}

// NewTransactionManager creates a new TransactionManager.
func NewTransactionManager(db *gorm.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

// RunInTransaction executes the given function within a database transaction.
// If the function returns an error, the transaction will be rolled back.
// If the function completes successfully, the transaction will be committed.
func (tm *TransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

// GetTx returns the transaction from context if available, otherwise returns the default DB.
func (tm *TransactionManager) GetTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return tm.db.WithContext(ctx)
}

// GetTxFromContext returns the transaction from context if available.
// This is a standalone function for use in repositories.
func GetTxFromContext(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return defaultDB.WithContext(ctx)
}
