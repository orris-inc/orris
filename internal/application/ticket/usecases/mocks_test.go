package usecases

import (
	"context"

	"orris/internal/domain/shared/events"
	"orris/internal/domain/ticket"
	"orris/internal/domain/user"
	"orris/internal/shared/logger"
)

type mockTicketRepository struct {
	SaveFunc             func(ctx context.Context, t *ticket.Ticket) error
	UpdateFunc           func(ctx context.Context, t *ticket.Ticket) error
	DeleteFunc           func(ctx context.Context, ticketID uint) error
	GetByIDFunc          func(ctx context.Context, ticketID uint) (*ticket.Ticket, error)
	GetByNumberFunc      func(ctx context.Context, number string) (*ticket.Ticket, error)
	ListFunc             func(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error)
	GetUserTicketsFunc   func(ctx context.Context, userID uint, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error)
	GetAssignedTicketsFunc func(ctx context.Context, assigneeID uint, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error)
	GetOverdueTicketsFunc func(ctx context.Context) ([]*ticket.Ticket, error)
}

func (m *mockTicketRepository) Save(ctx context.Context, t *ticket.Ticket) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, t)
	}
	return nil
}

func (m *mockTicketRepository) Update(ctx context.Context, t *ticket.Ticket) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, t)
	}
	return nil
}

func (m *mockTicketRepository) Delete(ctx context.Context, ticketID uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, ticketID)
	}
	return nil
}

func (m *mockTicketRepository) GetByID(ctx context.Context, ticketID uint) (*ticket.Ticket, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, ticketID)
	}
	return nil, nil
}

func (m *mockTicketRepository) GetByNumber(ctx context.Context, number string) (*ticket.Ticket, error) {
	if m.GetByNumberFunc != nil {
		return m.GetByNumberFunc(ctx, number)
	}
	return nil, nil
}

func (m *mockTicketRepository) List(ctx context.Context, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filters)
	}
	return nil, 0, nil
}

func (m *mockTicketRepository) GetUserTickets(ctx context.Context, userID uint, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
	if m.GetUserTicketsFunc != nil {
		return m.GetUserTicketsFunc(ctx, userID, filters)
	}
	return nil, 0, nil
}

func (m *mockTicketRepository) GetAssignedTickets(ctx context.Context, assigneeID uint, filters ticket.TicketFilter) ([]*ticket.Ticket, int64, error) {
	if m.GetAssignedTicketsFunc != nil {
		return m.GetAssignedTicketsFunc(ctx, assigneeID, filters)
	}
	return nil, 0, nil
}

func (m *mockTicketRepository) GetOverdueTickets(ctx context.Context) ([]*ticket.Ticket, error) {
	if m.GetOverdueTicketsFunc != nil {
		return m.GetOverdueTicketsFunc(ctx)
	}
	return nil, nil
}

type mockCommentRepository struct {
	SaveFunc         func(ctx context.Context, comment *ticket.Comment) error
	GetByTicketIDFunc func(ctx context.Context, ticketID uint) ([]*ticket.Comment, error)
	DeleteFunc       func(ctx context.Context, commentID uint) error
}

func (m *mockCommentRepository) Save(ctx context.Context, comment *ticket.Comment) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, comment)
	}
	return nil
}

func (m *mockCommentRepository) GetByTicketID(ctx context.Context, ticketID uint) ([]*ticket.Comment, error) {
	if m.GetByTicketIDFunc != nil {
		return m.GetByTicketIDFunc(ctx, ticketID)
	}
	return nil, nil
}

func (m *mockCommentRepository) Delete(ctx context.Context, commentID uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, commentID)
	}
	return nil
}

type mockEventDispatcher struct {
	PublishFunc    func(event events.DomainEvent) error
	PublishAllFunc func(events []events.DomainEvent) error
	StartFunc      func() error
	StopFunc       func() error
	SubscribeFunc  func(eventType string, handler events.EventHandler)
}

func (m *mockEventDispatcher) Publish(event events.DomainEvent) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(event)
	}
	return nil
}

func (m *mockEventDispatcher) PublishAll(evts []events.DomainEvent) error {
	if m.PublishAllFunc != nil {
		return m.PublishAllFunc(evts)
	}
	return nil
}

func (m *mockEventDispatcher) Start() error {
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

func (m *mockEventDispatcher) Stop() error {
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}

func (m *mockEventDispatcher) Subscribe(eventType string, handler events.EventHandler) error {
	if m.SubscribeFunc != nil {
		m.SubscribeFunc(eventType, handler)
	}
	return nil
}

func (m *mockEventDispatcher) Unsubscribe(eventType string, handler events.EventHandler) error {
	return nil
}

type mockLogger struct {
	DebugFunc  func(msg string, args ...any)
	InfoFunc   func(msg string, args ...any)
	WarnFunc   func(msg string, args ...any)
	ErrorFunc  func(msg string, args ...any)
	FatalFunc  func(msg string, args ...any)
	InfowFunc  func(msg string, keysAndValues ...interface{})
	ErrorwFunc func(msg string, keysAndValues ...interface{})
	WarnwFunc  func(msg string, keysAndValues ...interface{})
	DebugwFunc func(msg string, keysAndValues ...interface{})
	WithFunc   func(args ...any) interface{}
	NamedFunc  func(name string) interface{}
}

func (m *mockLogger) Debug(msg string, args ...any) {
	if m.DebugFunc != nil {
		m.DebugFunc(msg, args...)
	}
}

func (m *mockLogger) Info(msg string, args ...any) {
	if m.InfoFunc != nil {
		m.InfoFunc(msg, args...)
	}
}

func (m *mockLogger) Warn(msg string, args ...any) {
	if m.WarnFunc != nil {
		m.WarnFunc(msg, args...)
	}
}

func (m *mockLogger) Error(msg string, args ...any) {
	if m.ErrorFunc != nil {
		m.ErrorFunc(msg, args...)
	}
}

func (m *mockLogger) Fatal(msg string, args ...any) {
	if m.FatalFunc != nil {
		m.FatalFunc(msg, args...)
	}
}

func (m *mockLogger) With(args ...any) logger.Interface {
	if m.WithFunc != nil {
		if result, ok := m.WithFunc(args...).(logger.Interface); ok {
			return result
		}
	}
	return m
}

func (m *mockLogger) Named(name string) logger.Interface {
	if m.NamedFunc != nil {
		if result, ok := m.NamedFunc(name).(logger.Interface); ok {
			return result
		}
	}
	return m
}

func (m *mockLogger) Infow(msg string, keysAndValues ...interface{}) {
	if m.InfowFunc != nil {
		m.InfowFunc(msg, keysAndValues...)
	}
}

func (m *mockLogger) Errorw(msg string, keysAndValues ...interface{}) {
	if m.ErrorwFunc != nil {
		m.ErrorwFunc(msg, keysAndValues...)
	}
}

func (m *mockLogger) Warnw(msg string, keysAndValues ...interface{}) {
	if m.WarnwFunc != nil {
		m.WarnwFunc(msg, keysAndValues...)
	}
}

func (m *mockLogger) Debugw(msg string, keysAndValues ...interface{}) {
	if m.DebugwFunc != nil {
		m.DebugwFunc(msg, keysAndValues...)
	}
}

func (m *mockLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	if m.ErrorwFunc != nil {
		m.ErrorwFunc(msg, keysAndValues...)
	}
}

type mockUserRepository struct {
	CreateFunc                  func(ctx context.Context, u *user.User) error
	GetByIDFunc                 func(ctx context.Context, id uint) (*user.User, error)
	GetByEmailFunc              func(ctx context.Context, email string) (*user.User, error)
	UpdateFunc                  func(ctx context.Context, u *user.User) error
	DeleteFunc                  func(ctx context.Context, id uint) error
	ListFunc                    func(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error)
	ExistsFunc                  func(ctx context.Context, id uint) (bool, error)
	ExistsByEmailFunc           func(ctx context.Context, email string) (bool, error)
	GetByVerificationTokenFunc  func(ctx context.Context, token string) (*user.User, error)
	GetByPasswordResetTokenFunc func(ctx context.Context, token string) (*user.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, u *user.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, u)
	}
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uint) (*user.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, nil
}

func (m *mockUserRepository) Update(ctx context.Context, u *user.User) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, u)
	}
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uint) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, filter user.ListFilter) ([]*user.User, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockUserRepository) Exists(ctx context.Context, id uint) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, id)
	}
	return false, nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsByEmailFunc != nil {
		return m.ExistsByEmailFunc(ctx, email)
	}
	return false, nil
}

func (m *mockUserRepository) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	if m.GetByVerificationTokenFunc != nil {
		return m.GetByVerificationTokenFunc(ctx, token)
	}
	return nil, nil
}

func (m *mockUserRepository) GetByPasswordResetToken(ctx context.Context, token string) (*user.User, error) {
	if m.GetByPasswordResetTokenFunc != nil {
		return m.GetByPasswordResetTokenFunc(ctx, token)
	}
	return nil, nil
}
