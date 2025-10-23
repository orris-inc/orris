package usecases

import (
	"context"

	"orris/internal/domain/node"
	"orris/internal/domain/shared/events"
	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"

	"github.com/stretchr/testify/mock"
)

type mockEventDispatcher struct {
	mock.Mock
}

func (m *mockEventDispatcher) Publish(event events.DomainEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *mockEventDispatcher) PublishAll(eventList []events.DomainEvent) error {
	args := m.Called(eventList)
	return args.Error(0)
}

func (m *mockEventDispatcher) Subscribe(eventType string, handler events.EventHandler) error {
	args := m.Called(eventType, handler)
	return args.Error(0)
}

func (m *mockEventDispatcher) Unsubscribe(eventType string, handler events.EventHandler) error {
	args := m.Called(eventType, handler)
	return args.Error(0)
}

func (m *mockEventDispatcher) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockEventDispatcher) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockEventDispatcher) Dispatch(ctx context.Context, event interface{}) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockEventDispatcher) Register(eventType string, handler events.EventHandler) {
	m.Called(eventType, handler)
}

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Fatal(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) With(keysAndValues ...interface{}) logger.Interface {
	args := m.Called(keysAndValues)
	if args.Get(0) == nil {
		return m
	}
	return args.Get(0).(logger.Interface)
}

func (m *mockLogger) Named(name string) logger.Interface {
	args := m.Called(name)
	if args.Get(0) == nil {
		return m
	}
	return args.Get(0).(logger.Interface)
}

func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Infow(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}


type mockNodeGroupRepository struct {
	mock.Mock
}

func (m *mockNodeGroupRepository) Create(ctx context.Context, group *node.NodeGroup) error {
	args := m.Called(ctx, group)
	if args.Get(0) == nil {
		if group.ID() == 0 {
			_ = group.SetID(1)
		}
	}
	return args.Error(0)
}

func (m *mockNodeGroupRepository) GetByID(ctx context.Context, id uint) (*node.NodeGroup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*node.NodeGroup), args.Error(1)
}

func (m *mockNodeGroupRepository) GetByName(ctx context.Context, name string) (*node.NodeGroup, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*node.NodeGroup), args.Error(1)
}

func (m *mockNodeGroupRepository) Update(ctx context.Context, group *node.NodeGroup) error {
	args := m.Called(ctx, group)
	return args.Error(0)
}

func (m *mockNodeGroupRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockNodeGroupRepository) List(ctx context.Context, filter node.NodeGroupFilter) ([]*node.NodeGroup, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*node.NodeGroup), args.Get(1).(int64), args.Error(2)
}

func (m *mockNodeGroupRepository) GetPublicGroups(ctx context.Context) ([]*node.NodeGroup, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeGroup), args.Error(1)
}

func (m *mockNodeGroupRepository) GetBySubscriptionPlanID(ctx context.Context, planID uint) ([]*node.NodeGroup, error) {
	args := m.Called(ctx, planID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.NodeGroup), args.Error(1)
}

func (m *mockNodeGroupRepository) AddNode(ctx context.Context, groupID, nodeID uint) error {
	args := m.Called(ctx, groupID, nodeID)
	return args.Error(0)
}

func (m *mockNodeGroupRepository) RemoveNode(ctx context.Context, groupID, nodeID uint) error {
	args := m.Called(ctx, groupID, nodeID)
	return args.Error(0)
}

func (m *mockNodeGroupRepository) GetNodesByGroupID(ctx context.Context, groupID uint) ([]*node.Node, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.Node), args.Error(1)
}

func (m *mockNodeGroupRepository) AssociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error {
	args := m.Called(ctx, groupID, planID)
	return args.Error(0)
}

func (m *mockNodeGroupRepository) DisassociateSubscriptionPlan(ctx context.Context, groupID, planID uint) error {
	args := m.Called(ctx, groupID, planID)
	return args.Error(0)
}

func (m *mockNodeGroupRepository) GetSubscriptionPlansByGroupID(ctx context.Context, groupID uint) ([]uint, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint), args.Error(1)
}

func (m *mockNodeGroupRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

type mockSubscriptionPlanRepository struct {
	mock.Mock
}

func (m *mockSubscriptionPlanRepository) Create(ctx context.Context, plan *subscription.SubscriptionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *mockSubscriptionPlanRepository) GetByID(ctx context.Context, id uint) (*subscription.SubscriptionPlan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.SubscriptionPlan), args.Error(1)
}

func (m *mockSubscriptionPlanRepository) GetBySlug(ctx context.Context, slug string) (*subscription.SubscriptionPlan, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*subscription.SubscriptionPlan), args.Error(1)
}

func (m *mockSubscriptionPlanRepository) Update(ctx context.Context, plan *subscription.SubscriptionPlan) error {
	args := m.Called(ctx, plan)
	return args.Error(0)
}

func (m *mockSubscriptionPlanRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockSubscriptionPlanRepository) GetActivePublicPlans(ctx context.Context) ([]*subscription.SubscriptionPlan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*subscription.SubscriptionPlan), args.Error(1)
}

func (m *mockSubscriptionPlanRepository) GetAllActive(ctx context.Context) ([]*subscription.SubscriptionPlan, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*subscription.SubscriptionPlan), args.Error(1)
}

func (m *mockSubscriptionPlanRepository) List(ctx context.Context, filter subscription.PlanFilter) ([]*subscription.SubscriptionPlan, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*subscription.SubscriptionPlan), args.Get(1).(int64), args.Error(2)
}

func (m *mockSubscriptionPlanRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

type mockNodeRepository struct {
	mock.Mock
}

func (m *mockNodeRepository) Create(ctx context.Context, n *node.Node) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *mockNodeRepository) GetByID(ctx context.Context, id uint) (*node.Node, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*node.Node), args.Error(1)
}

func (m *mockNodeRepository) GetByToken(ctx context.Context, tokenHash string) (*node.Node, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*node.Node), args.Error(1)
}

func (m *mockNodeRepository) Update(ctx context.Context, n *node.Node) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

func (m *mockNodeRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockNodeRepository) List(ctx context.Context, filter node.NodeFilter) ([]*node.Node, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*node.Node), args.Get(1).(int64), args.Error(2)
}

func (m *mockNodeRepository) GetByGroupID(ctx context.Context, groupID uint) ([]*node.Node, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.Node), args.Error(1)
}

func (m *mockNodeRepository) GetByStatus(ctx context.Context, status string) ([]*node.Node, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.Node), args.Error(1)
}

func (m *mockNodeRepository) GetAvailableNodes(ctx context.Context) ([]*node.Node, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*node.Node), args.Error(1)
}

func (m *mockNodeRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *mockNodeRepository) ExistsByAddress(ctx context.Context, address string, port int) (bool, error) {
	args := m.Called(ctx, address, port)
	return args.Bool(0), args.Error(1)
}
