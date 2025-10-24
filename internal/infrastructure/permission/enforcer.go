package permission

import (
	"fmt"
	"sync"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"

	"orris/internal/domain/permission"
	"orris/internal/shared/logger"
)

var _ permission.PermissionEnforcer = (*Enforcer)(nil)

type Enforcer struct {
	enforcer *casbin.Enforcer
	mu       sync.RWMutex
	logger   logger.Interface
}

func NewEnforcer(db *gorm.DB, modelPath string, log logger.Interface) (*Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	return &Enforcer{
		enforcer: enforcer,
		logger:   log,
	}, nil
}

func (e *Enforcer) Enforce(userID string, resource string, action string) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	allowed, err := e.enforcer.Enforce(userID, resource, action)
	if err != nil {
		e.logger.Errorw("permission check failed", "error", err, "user_id", userID, "resource", resource, "action", action)
		return false, fmt.Errorf("permission check failed: %w", err)
	}

	return allowed, nil
}

func (e *Enforcer) AddPolicy(role string, resource string, action string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := e.enforcer.AddPolicy(role, resource, action)
	if err != nil {
		e.logger.Errorw("failed to add policy", "error", err)
		return fmt.Errorf("failed to add policy: %w", err)
	}

	return e.enforcer.SavePolicy()
}

func (e *Enforcer) RemovePolicy(role string, resource string, action string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := e.enforcer.RemovePolicy(role, resource, action)
	if err != nil {
		e.logger.Errorw("failed to remove policy", "error", err)
		return fmt.Errorf("failed to remove policy: %w", err)
	}

	return e.enforcer.SavePolicy()
}

func (e *Enforcer) AddRoleForUser(userID string, role string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := e.enforcer.AddRoleForUser(userID, role)
	if err != nil {
		e.logger.Errorw("failed to add role for user", "error", err, "user_id", userID, "role", role)
		return fmt.Errorf("failed to add role for user: %w", err)
	}

	return e.enforcer.SavePolicy()
}

func (e *Enforcer) DeleteRoleForUser(userID string, role string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := e.enforcer.DeleteRoleForUser(userID, role)
	if err != nil {
		e.logger.Errorw("failed to delete role for user", "error", err, "user_id", userID, "role", role)
		return fmt.Errorf("failed to delete role for user: %w", err)
	}

	return e.enforcer.SavePolicy()
}

func (e *Enforcer) GetRolesForUser(userID string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	roles, err := e.enforcer.GetRolesForUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles for user: %w", err)
	}

	return roles, nil
}

func (e *Enforcer) GetPermissionsForUser(userID string) ([][]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	permissions, err := e.enforcer.GetImplicitPermissionsForUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions for user: %w", err)
	}

	return permissions, nil
}

func (e *Enforcer) LoadPolicy() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("failed to reload policy: %w", err)
	}

	e.logger.Info("policy reloaded successfully")
	return nil
}
