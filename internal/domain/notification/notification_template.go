package notification

import (
	"bytes"
	"fmt"
	"sync"
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
	mu           sync.RWMutex
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
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.id
}

func (t *NotificationTemplate) TemplateType() vo.TemplateType {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.templateType
}

func (t *NotificationTemplate) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

func (t *NotificationTemplate) Title() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.title
}

func (t *NotificationTemplate) Content() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.content
}

func (t *NotificationTemplate) Variables() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	vars := make([]string, len(t.variables))
	copy(vars, t.variables)
	return vars
}

func (t *NotificationTemplate) Enabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}

func (t *NotificationTemplate) Version() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.version
}

func (t *NotificationTemplate) CreatedAt() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.createdAt
}

func (t *NotificationTemplate) UpdatedAt() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.updatedAt
}

func (t *NotificationTemplate) SetID(id uint) error {
	t.mu.Lock()
	defer t.mu.Unlock()

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
	t.mu.RLock()
	defer t.mu.RUnlock()

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
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.enabled {
		return
	}

	t.enabled = true
	t.updatedAt = time.Now()
	t.version++
}

func (t *NotificationTemplate) Disable() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.enabled {
		return
	}

	t.enabled = false
	t.updatedAt = time.Now()
	t.version++
}

func (t *NotificationTemplate) Update(name, title, content string, variables []string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

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
