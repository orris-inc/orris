package permission

import (
	"fmt"

	"gorm.io/gorm"

	"orris/internal/shared/logger"
)

type PermissionSync struct {
	db     *gorm.DB
	logger logger.Interface
}

func NewPermissionSync(db *gorm.DB, logger logger.Interface) *PermissionSync {
	return &PermissionSync{
		db:     db,
		logger: logger,
	}
}

func (s *PermissionSync) SyncToCasbin() error {
	s.logger.Info("syncing permissions to Casbin...")

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := s.syncRolePermissions(tx); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to sync role permissions: %w", err)
	}

	if err := s.syncUserRoles(tx); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to sync user roles: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("permissions synced to Casbin successfully")
	return nil
}

func (s *PermissionSync) syncRolePermissions(tx *gorm.DB) error {
	query := `
		INSERT INTO casbin_rule (ptype, v0, v1, v2)
		SELECT DISTINCT
			'p',
			r.slug,
			p.resource,
			p.action
		FROM role_permissions rp
		JOIN roles r ON rp.role_id = r.id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE NOT EXISTS (
			SELECT 1 FROM casbin_rule cr
			WHERE cr.ptype = 'p'
			AND cr.v0 = r.slug
			AND cr.v1 = p.resource
			AND cr.v2 = p.action
		)
	`

	result := tx.Exec(query)
	if result.Error != nil {
		return fmt.Errorf("failed to sync role permissions: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		s.logger.Infow("synced role permissions to Casbin", "count", result.RowsAffected)
	}

	return nil
}

func (s *PermissionSync) syncUserRoles(tx *gorm.DB) error {
	query := `
		INSERT INTO casbin_rule (ptype, v0, v1, v2)
		SELECT DISTINCT
			'g',
			CAST(ur.user_id AS CHAR),
			r.slug,
			''
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE NOT EXISTS (
			SELECT 1 FROM casbin_rule cr
			WHERE cr.ptype = 'g'
			AND cr.v0 = CAST(ur.user_id AS CHAR)
			AND cr.v1 = r.slug
		)
	`

	result := tx.Exec(query)
	if result.Error != nil {
		return fmt.Errorf("failed to sync user roles: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		s.logger.Infow("synced user roles to Casbin", "count", result.RowsAffected)
	}

	return nil
}
