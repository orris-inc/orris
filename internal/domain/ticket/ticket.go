package ticket

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/ticket/value_objects"
)

type Ticket struct {
	id           uint
	number       string
	title        string
	description  string
	category     vo.Category
	priority     vo.Priority
	status       vo.TicketStatus
	creatorID    uint
	assigneeID   *uint
	tags         []string
	metadata     map[string]interface{}
	slaDueTime   *time.Time
	responseTime *time.Time
	resolvedTime *time.Time
	version      int
	createdAt    time.Time
	updatedAt    time.Time
	closedAt     *time.Time
	comments     []*Comment
}

func NewTicket(
	title string,
	description string,
	category vo.Category,
	priority vo.Priority,
	creatorID uint,
) (*Ticket, error) {
	if len(title) == 0 {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return nil, fmt.Errorf("title exceeds maximum length of 200 characters")
	}
	if len(description) == 0 {
		return nil, fmt.Errorf("description is required")
	}
	if len(description) > 5000 {
		return nil, fmt.Errorf("description exceeds maximum length of 5000 characters")
	}
	if !category.IsValid() {
		return nil, fmt.Errorf("invalid category")
	}
	if !priority.IsValid() {
		return nil, fmt.Errorf("invalid priority")
	}
	if creatorID == 0 {
		return nil, fmt.Errorf("creator ID is required")
	}

	now := time.Now()
	slaDueTime := now.Add(time.Duration(priority.GetSLAHours()) * time.Hour)

	t := &Ticket{
		title:       title,
		description: description,
		category:    category,
		priority:    priority,
		status:      vo.StatusNew,
		creatorID:   creatorID,
		tags:        []string{},
		metadata:    make(map[string]interface{}),
		slaDueTime:  &slaDueTime,
		version:     1,
		createdAt:   now,
		updatedAt:   now,
		comments:    []*Comment{},
	}

	return t, nil
}

func ReconstructTicket(
	id uint,
	number string,
	title string,
	description string,
	category vo.Category,
	priority vo.Priority,
	status vo.TicketStatus,
	creatorID uint,
	assigneeID *uint,
	tags []string,
	metadata map[string]interface{},
	slaDueTime *time.Time,
	responseTime *time.Time,
	resolvedTime *time.Time,
	version int,
	createdAt, updatedAt time.Time,
	closedAt *time.Time,
) (*Ticket, error) {
	if id == 0 {
		return nil, fmt.Errorf("ticket ID cannot be zero")
	}
	if len(number) == 0 {
		return nil, fmt.Errorf("ticket number is required")
	}
	if len(title) == 0 {
		return nil, fmt.Errorf("title is required")
	}
	if !category.IsValid() {
		return nil, fmt.Errorf("invalid category")
	}
	if !priority.IsValid() {
		return nil, fmt.Errorf("invalid priority")
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid status")
	}

	if tags == nil {
		tags = []string{}
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Ticket{
		id:           id,
		number:       number,
		title:        title,
		description:  description,
		category:     category,
		priority:     priority,
		status:       status,
		creatorID:    creatorID,
		assigneeID:   assigneeID,
		tags:         tags,
		metadata:     metadata,
		slaDueTime:   slaDueTime,
		responseTime: responseTime,
		resolvedTime: resolvedTime,
		version:      version,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		closedAt:     closedAt,
		comments:     []*Comment{},
	}, nil
}

func (t *Ticket) ID() uint {
	return t.id
}

func (t *Ticket) Number() string {
	return t.number
}

func (t *Ticket) Title() string {
	return t.title
}

func (t *Ticket) Description() string {
	return t.description
}

func (t *Ticket) Category() vo.Category {
	return t.category
}

func (t *Ticket) Priority() vo.Priority {
	return t.priority
}

func (t *Ticket) Status() vo.TicketStatus {
	return t.status
}

func (t *Ticket) CreatorID() uint {
	return t.creatorID
}

func (t *Ticket) AssigneeID() *uint {
	return t.assigneeID
}

func (t *Ticket) Tags() []string {
	tagsCopy := make([]string, len(t.tags))
	copy(tagsCopy, t.tags)
	return tagsCopy
}

func (t *Ticket) Metadata() map[string]interface{} {
	metadataCopy := make(map[string]interface{})
	for k, v := range t.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

func (t *Ticket) SLADueTime() *time.Time {
	return t.slaDueTime
}

func (t *Ticket) ResponseTime() *time.Time {
	return t.responseTime
}

func (t *Ticket) ResolvedTime() *time.Time {
	return t.resolvedTime
}

func (t *Ticket) Version() int {
	return t.version
}

func (t *Ticket) CreatedAt() time.Time {
	return t.createdAt
}

func (t *Ticket) UpdatedAt() time.Time {
	return t.updatedAt
}

func (t *Ticket) ClosedAt() *time.Time {
	return t.closedAt
}

func (t *Ticket) Comments() []*Comment {
	commentsCopy := make([]*Comment, len(t.comments))
	copy(commentsCopy, t.comments)
	return commentsCopy
}

func (t *Ticket) SetID(id uint) error {
	if t.id != 0 {
		return fmt.Errorf("ticket ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("ticket ID cannot be zero")
	}
	t.id = id
	return nil
}

func (t *Ticket) SetNumber(number string) error {
	if len(t.number) > 0 {
		return fmt.Errorf("ticket number is already set")
	}
	if len(number) == 0 {
		return fmt.Errorf("ticket number cannot be empty")
	}
	t.number = number
	return nil
}

func (t *Ticket) AssignTo(assigneeID uint, assignedBy uint) error {
	if assigneeID == 0 {
		return fmt.Errorf("assignee ID cannot be zero")
	}

	t.assigneeID = &assigneeID
	t.updatedAt = time.Now()
	t.version++

	if t.status.IsNew() {
		t.status = vo.StatusOpen
	}

	return nil
}

func (t *Ticket) ChangeStatus(newStatus vo.TicketStatus, changedBy uint) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("invalid status: %s", newStatus)
	}

	if t.status == newStatus {
		return nil
	}

	if !t.status.CanTransitionTo(newStatus) {
		return fmt.Errorf("cannot transition from %s to %s", t.status, newStatus)
	}

	t.status = newStatus
	t.updatedAt = time.Now()
	t.version++

	if newStatus.IsResolved() && t.resolvedTime == nil {
		now := time.Now()
		t.resolvedTime = &now
	}

	if newStatus.IsClosed() && t.closedAt == nil {
		now := time.Now()
		t.closedAt = &now
	}

	if newStatus.IsReopened() {
		t.closedAt = nil
		t.resolvedTime = nil
	}

	return nil
}

func (t *Ticket) ChangePriority(newPriority vo.Priority, changedBy uint) error {
	if !newPriority.IsValid() {
		return fmt.Errorf("invalid priority: %s", newPriority)
	}

	if t.priority == newPriority {
		return nil
	}

	t.priority = newPriority
	t.updatedAt = time.Now()
	t.version++

	if !t.createdAt.IsZero() {
		newSLADueTime := t.createdAt.Add(time.Duration(newPriority.GetSLAHours()) * time.Hour)
		t.slaDueTime = &newSLADueTime
	}

	return nil
}

func (t *Ticket) AddComment(comment *Comment) error {
	if comment == nil {
		return fmt.Errorf("comment cannot be nil")
	}

	if comment.TicketID() != t.id {
		return fmt.Errorf("comment ticket ID mismatch")
	}

	t.comments = append(t.comments, comment)
	t.updatedAt = time.Now()

	if t.responseTime == nil && !comment.IsInternal() {
		if comment.UserID() != t.creatorID {
			now := time.Now()
			t.responseTime = &now
		}
	}

	return nil
}

func (t *Ticket) Close(reason string, closedBy uint) error {
	if len(reason) == 0 {
		return fmt.Errorf("close reason is required")
	}

	if t.status.IsClosed() {
		return nil
	}

	if !t.status.CanTransitionTo(vo.StatusClosed) {
		return fmt.Errorf("cannot close ticket with status %s", t.status)
	}

	t.status = vo.StatusClosed
	now := time.Now()
	t.closedAt = &now
	t.updatedAt = now
	t.version++

	return nil
}

func (t *Ticket) Reopen(reason string, reopenedBy uint) error {
	if len(reason) == 0 {
		return fmt.Errorf("reopen reason is required")
	}

	if !t.status.IsClosed() && !t.status.IsResolved() {
		return fmt.Errorf("only closed or resolved tickets can be reopened")
	}

	t.status = vo.StatusReopened
	t.closedAt = nil
	t.resolvedTime = nil
	t.updatedAt = time.Now()
	t.version++

	return nil
}

func (t *Ticket) IsOverdue() bool {
	if t.slaDueTime == nil {
		return false
	}

	if t.status.IsClosed() || t.status.IsResolved() {
		return false
	}

	return time.Now().After(*t.slaDueTime)
}

func (t *Ticket) MarkFirstResponse() error {
	if t.responseTime != nil {
		return fmt.Errorf("first response already marked")
	}

	now := time.Now()
	t.responseTime = &now
	t.updatedAt = now

	return nil
}

func (t *Ticket) MarkResolved() error {
	if t.resolvedTime != nil {
		return fmt.Errorf("ticket already marked as resolved")
	}

	now := time.Now()
	t.resolvedTime = &now
	t.updatedAt = now

	return nil
}

func (t *Ticket) CanBeViewedBy(userID uint, userRoles []string) bool {
	for _, role := range userRoles {
		if role == "admin" || role == "support_agent" {
			return true
		}
	}

	if t.creatorID == userID {
		return true
	}

	if t.assigneeID != nil && *t.assigneeID == userID {
		return true
	}

	return false
}

func (t *Ticket) Validate() error {
	if len(t.title) == 0 {
		return fmt.Errorf("title is required")
	}
	if len(t.description) == 0 {
		return fmt.Errorf("description is required")
	}
	if !t.category.IsValid() {
		return fmt.Errorf("invalid category")
	}
	if !t.priority.IsValid() {
		return fmt.Errorf("invalid priority")
	}
	if !t.status.IsValid() {
		return fmt.Errorf("invalid status")
	}
	if t.creatorID == 0 {
		return fmt.Errorf("creator ID is required")
	}
	return nil
}
