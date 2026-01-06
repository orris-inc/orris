package admin

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminNotifier defines the interface for sending admin notifications
// This interface is used by other use cases to trigger admin notifications
type AdminNotifier interface {
	// NotifyNewUser sends a new user registration notification to admins
	NotifyNewUser(ctx context.Context, cmd NotifyNewUserCommand) error

	// NotifyPaymentSuccess sends a payment success notification to admins
	NotifyPaymentSuccess(ctx context.Context, cmd NotifyPaymentSuccessCommand) error

	// NotifyNodeOnline sends a node online notification to admins
	NotifyNodeOnline(ctx context.Context, cmd NotifyNodeOnlineCommand) error

	// NotifyNodeOffline sends a node offline notification to admins
	NotifyNodeOffline(ctx context.Context, cmd NotifyNodeOfflineCommand) error

	// NotifyAgentOnline sends an agent online notification to admins
	NotifyAgentOnline(ctx context.Context, cmd NotifyAgentOnlineCommand) error

	// NotifyAgentOffline sends an agent offline notification to admins
	NotifyAgentOffline(ctx context.Context, cmd NotifyAgentOfflineCommand) error
}

// NotifyNewUserCommand contains data for new user notification
type NotifyNewUserCommand struct {
	UserID    uint
	UserSID   string
	Email     string
	Name      string
	Source    string // e.g., "registration", "oauth"
	CreatedAt time.Time
}

// NotifyPaymentSuccessCommand contains data for payment success notification
type NotifyPaymentSuccessCommand struct {
	PaymentID      uint
	PaymentSID     string
	UserID         uint
	UserSID        string
	UserEmail      string
	SubscriptionID uint
	PlanName       string
	Amount         float64 // In main currency unit (e.g., 99.00)
	Currency       string  // e.g., "CNY", "USD"
	PaymentMethod  string  // e.g., "alipay", "wechat", "stripe"
	TransactionID  string
	PaidAt         time.Time
}

// NotifyNodeOnlineCommand contains data for node online notification
type NotifyNodeOnlineCommand struct {
	NodeID           uint
	NodeSID          string
	NodeName         string
	MuteNotification bool // if true, skip sending notification
}

// NotifyNodeOfflineCommand contains data for node offline notification
type NotifyNodeOfflineCommand struct {
	NodeID           uint
	NodeSID          string
	NodeName         string
	LastSeenAt       time.Time
	OfflineMinutes   int
	MuteNotification bool // if true, skip sending notification
}

// NotifyAgentOnlineCommand contains data for agent online notification
type NotifyAgentOnlineCommand struct {
	AgentID          uint
	AgentSID         string
	AgentName        string
	MuteNotification bool // if true, skip sending notification
}

// NotifyAgentOfflineCommand contains data for agent offline notification
type NotifyAgentOfflineCommand struct {
	AgentID          uint
	AgentSID         string
	AgentName        string
	LastSeenAt       time.Time
	OfflineMinutes   int
	MuteNotification bool // if true, skip sending notification
}

// NoopAdminNotifier is a no-op implementation of AdminNotifier
// Used when admin notification is not configured
type NoopAdminNotifier struct {
	logger logger.Interface
}

// NewNoopAdminNotifier creates a new NoopAdminNotifier
func NewNoopAdminNotifier(logger logger.Interface) *NoopAdminNotifier {
	return &NoopAdminNotifier{logger: logger}
}

func (n *NoopAdminNotifier) NotifyNewUser(ctx context.Context, cmd NotifyNewUserCommand) error {
	n.logger.Debugw("admin notification skipped (not configured)", "type", "new_user", "user_sid", cmd.UserSID)
	return nil
}

func (n *NoopAdminNotifier) NotifyPaymentSuccess(ctx context.Context, cmd NotifyPaymentSuccessCommand) error {
	n.logger.Debugw("admin notification skipped (not configured)", "type", "payment_success", "payment_sid", cmd.PaymentSID)
	return nil
}

func (n *NoopAdminNotifier) NotifyNodeOnline(ctx context.Context, cmd NotifyNodeOnlineCommand) error {
	n.logger.Debugw("admin notification skipped (not configured)", "type", "node_online", "node_sid", cmd.NodeSID)
	return nil
}

func (n *NoopAdminNotifier) NotifyNodeOffline(ctx context.Context, cmd NotifyNodeOfflineCommand) error {
	n.logger.Debugw("admin notification skipped (not configured)", "type", "node_offline", "node_sid", cmd.NodeSID)
	return nil
}

func (n *NoopAdminNotifier) NotifyAgentOnline(ctx context.Context, cmd NotifyAgentOnlineCommand) error {
	n.logger.Debugw("admin notification skipped (not configured)", "type", "agent_online", "agent_sid", cmd.AgentSID)
	return nil
}

func (n *NoopAdminNotifier) NotifyAgentOffline(ctx context.Context, cmd NotifyAgentOfflineCommand) error {
	n.logger.Debugw("admin notification skipped (not configured)", "type", "agent_offline", "agent_sid", cmd.AgentSID)
	return nil
}
