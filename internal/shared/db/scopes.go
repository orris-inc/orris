// Package db provides database utilities including transaction management and query scopes.
package db

import (
	"gorm.io/gorm"
)

// NotDeleted is a GORM scope that filters out soft-deleted records.
// Use this scope when querying with Model().Where().Count() or similar patterns
// that may not automatically apply soft delete filtering.
//
// Example usage:
//
//	db.Model(&Model{}).Scopes(db.NotDeleted()).Where("name = ?", name).Count(&count)
//	db.Table("table_name").Scopes(db.NotDeleted()).Where("id = ?", id).Find(&results)
func NotDeleted() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL")
	}
}

// NotDeletedWithAlias is a GORM scope that filters out soft-deleted records with a table alias.
// Use this when joining tables and need to specify which table's deleted_at to check.
//
// Example usage:
//
//	db.Table("users u").Scopes(db.NotDeletedWithAlias("u")).Find(&results)
func NotDeletedWithAlias(alias string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(alias + ".deleted_at IS NULL")
	}
}

// Active is a combined scope that filters for active (not deleted) records.
// This is an alias for NotDeleted() for semantic clarity.
func Active() func(db *gorm.DB) *gorm.DB {
	return NotDeleted()
}
