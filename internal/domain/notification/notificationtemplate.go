package notification

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	vo "orris/internal/domain/notification/value_objects"
)

type NotificationTemplate struct {
	id           uint
	templateType vo.TemplateType
	name         string
	title        string
	content      string
	variables    []string
	enabled      bool
	version      int
	createdAt    time.Time
	updatedAt    time.Time
}

func NewNotificationTemplate(
	templateType vo.TemplateType,
	name string,
	title string,
	content string,
	variables []string,
) (*NotificationTemplate, error) {
	if !templateType.IsValid() {
		return nil, fmt.Errorf("invalid template type")
	}
	if len(name) == 0 {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("name exceeds maximum length of 100 characters")
	}
	if len(title) == 0 {
		return nil, fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return nil, fmt.Errorf("title exceeds maximum length of 200 characters")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("content is required")
	}
	if len(content) > 10000 {
		return nil, fmt.Errorf("content exceeds maximum length of 10000 characters")
	}

	if variables == nil {
		variables = []string{}
	}

	now := time.Now()
	return &NotificationTemplate{
		templateType: templateType,
		name:         name,
		title:        title,
		content:      content,
		variables:    variables,
		enabled:      true,
		version:      1,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func ReconstructNotificationTemplate(
	id uint,
	templateType vo.TemplateType,
	name string,
	title string,
	content string,
	variables []string,
	enabled bool,
	version int,
	createdAt, updatedAt time.Time,
) (*NotificationTemplate, error) {
	if id == 0 {
		return nil, fmt.Errorf("template ID cannot be zero")
	}
	if !templateType.IsValid() {
		return nil, fmt.Errorf("invalid template type")
	}
	if len(name) == 0 {
		return nil, fmt.Errorf("name is required")
	}

	if variables == nil {
		variables = []string{}
	}

	return &NotificationTemplate{
		id:           id,
		templateType: templateType,
		name:         name,
		title:        title,
		content:      content,
		variables:    variables,
		enabled:      enabled,
		version:      version,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}

func (t *NotificationTemplate) ID() uint {
	return t.id
}

func (t *NotificationTemplate) TemplateType() vo.TemplateType {
	return t.templateType
}

func (t *NotificationTemplate) Name() string {
	return t.name
}

func (t *NotificationTemplate) Title() string {
	return t.title
}

func (t *NotificationTemplate) Content() string {
	return t.content
}

func (t *NotificationTemplate) Variables() []string {
	vars := make([]string, len(t.variables))
	copy(vars, t.variables)
	return vars
}

func (t *NotificationTemplate) Enabled() bool {
	return t.enabled
}

func (t *NotificationTemplate) Version() int {
	return t.version
}

func (t *NotificationTemplate) CreatedAt() time.Time {
	return t.createdAt
}

func (t *NotificationTemplate) UpdatedAt() time.Time {
	return t.updatedAt
}

func (t *NotificationTemplate) SetID(id uint) error {
	if t.id != 0 {
		return fmt.Errorf("template ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("template ID cannot be zero")
	}
	t.id = id
	return nil
}

func (t *NotificationTemplate) Render(data map[string]interface{}) (string, string, error) {
	if !t.enabled {
		return "", "", fmt.Errorf("template is disabled")
	}

	titleTmpl, err := template.New("title").Parse(t.title)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse title template: %w", err)
	}

	contentTmpl, err := template.New("content").Parse(t.content)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse content template: %w", err)
	}

	var titleBuf bytes.Buffer
	if err := titleTmpl.Execute(&titleBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute title template: %w", err)
	}

	var contentBuf bytes.Buffer
	if err := contentTmpl.Execute(&contentBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute content template: %w", err)
	}

	return titleBuf.String(), contentBuf.String(), nil
}

func (t *NotificationTemplate) Enable() {
	if t.enabled {
		return
	}

	t.enabled = true
	t.updatedAt = time.Now()
	t.version++
}

func (t *NotificationTemplate) Disable() {
	if !t.enabled {
		return
	}

	t.enabled = false
	t.updatedAt = time.Now()
	t.version++
}

func (t *NotificationTemplate) Update(name, title, content string, variables []string) error {
	if len(name) == 0 {
		return fmt.Errorf("name is required")
	}
	if len(name) > 100 {
		return fmt.Errorf("name exceeds maximum length of 100 characters")
	}
	if len(title) == 0 {
		return fmt.Errorf("title is required")
	}
	if len(title) > 200 {
		return fmt.Errorf("title exceeds maximum length of 200 characters")
	}
	if len(content) == 0 {
		return fmt.Errorf("content is required")
	}
	if len(content) > 10000 {
		return fmt.Errorf("content exceeds maximum length of 10000 characters")
	}

	if _, err := template.New("title").Parse(title); err != nil {
		return fmt.Errorf("invalid title template syntax: %w", err)
	}
	if _, err := template.New("content").Parse(content); err != nil {
		return fmt.Errorf("invalid content template syntax: %w", err)
	}

	if variables == nil {
		variables = []string{}
	}

	t.name = name
	t.title = title
	t.content = content
	t.variables = variables
	t.updatedAt = time.Now()
	t.version++

	return nil
}
