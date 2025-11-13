package models

type TicketModel struct {
	ID           uint   `gorm:"primaryKey"`
	Number       string `gorm:"uniqueIndex;size:50;not null"`
	Title        string `gorm:"size:200;not null"`
	Description  string `gorm:"type:text;not null"`
	Category     string `gorm:"size:50;not null;index"`
	Priority     string `gorm:"size:20;not null;index"`
	Status       string `gorm:"size:20;not null;index"`
	CreatorID    uint   `gorm:"not null;index"`
	AssigneeID   *uint  `gorm:"index"`
	Tags         string `gorm:"type:json"`
	Metadata     string `gorm:"type:json"`
	SLADueTime   *int64 `gorm:"index"`
	ResponseTime *int64
	ResolvedTime *int64
	Version      int   `gorm:"not null;default:1"`
	CreatedAt    int64 `gorm:"autoCreateTime:milli;not null"`
	UpdatedAt    int64 `gorm:"autoUpdateTime:milli;not null"`
	ClosedAt     *int64

	// Note: No foreign key constraints or associations.
	// All relationships are managed by application business logic.
}

func (TicketModel) TableName() string {
	return "tickets"
}

type CommentModel struct {
	ID         uint   `gorm:"primaryKey"`
	TicketID   uint   `gorm:"not null;index"`
	UserID     uint   `gorm:"not null;index"`
	Content    string `gorm:"type:text;not null"`
	IsInternal bool   `gorm:"not null;default:false"`
	CreatedAt  int64  `gorm:"autoCreateTime:milli;not null;index"`
	UpdatedAt  int64  `gorm:"autoUpdateTime:milli;not null"`
}

func (CommentModel) TableName() string {
	return "ticket_comments"
}
