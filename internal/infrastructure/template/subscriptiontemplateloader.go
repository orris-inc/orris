package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionTemplateLoader loads and manages subscription template files
type SubscriptionTemplateLoader struct {
	templates map[string]string // format -> file content
	path      string
	logger    logger.Interface
}

// NewSubscriptionTemplateLoader creates a new template loader
func NewSubscriptionTemplateLoader(path string, logger logger.Interface) *SubscriptionTemplateLoader {
	return &SubscriptionTemplateLoader{
		templates: make(map[string]string),
		path:      path,
		logger:    logger,
	}
}

// Load loads all templates from the configured directory
// Template files are named: custom.{format}.yaml or custom.{format}.conf
// Supported formats: clash, surge, v2ray, sip008, base64
func (l *SubscriptionTemplateLoader) Load() error {
	// Check if templates directory exists
	if _, err := os.Stat(l.path); os.IsNotExist(err) {
		l.logger.Warnw("templates directory not found, using default formatters", "path", l.path)
		return nil // Not an error - will fall back to defaults
	}

	// Supported template file patterns
	formats := []string{"clash", "surge", "v2ray", "sip008", "base64"}
	extensions := []string{".yaml", ".yml", ".conf"}

	for _, format := range formats {
		loaded := false
		for _, ext := range extensions {
			filename := fmt.Sprintf("custom.%s%s", format, ext)
			filePath := filepath.Join(l.path, filename)

			content, err := os.ReadFile(filePath)
			if err != nil {
				if !os.IsNotExist(err) {
					l.logger.Warnw("failed to read template file",
						"file", filePath,
						"error", err,
					)
				}
				continue
			}

			l.templates[format] = string(content)
			l.logger.Infow("loaded subscription template",
				"format", format,
				"file", filename,
				"size", len(content),
			)
			loaded = true
			break // Found template for this format
		}

		if !loaded {
			l.logger.Debugw("no custom template found for format, will use default", "format", format)
		}
	}

	if len(l.templates) == 0 {
		l.logger.Warnw("no custom templates loaded, using default formatters")
	} else {
		l.logger.Infow("subscription templates loaded", "count", len(l.templates))
	}

	return nil
}

// Get returns the template content for a given format
// Returns (content, true) if template exists, ("", false) otherwise
func (l *SubscriptionTemplateLoader) Get(formatType string) (string, bool) {
	// Normalize format type
	formatType = strings.ToLower(strings.TrimSpace(formatType))

	content, ok := l.templates[formatType]
	return content, ok
}

// HasTemplate checks if a custom template exists for the given format
func (l *SubscriptionTemplateLoader) HasTemplate(formatType string) bool {
	_, ok := l.Get(formatType)
	return ok
}

// GetLoadedFormats returns a list of formats that have loaded templates
func (l *SubscriptionTemplateLoader) GetLoadedFormats() []string {
	formats := make([]string, 0, len(l.templates))
	for format := range l.templates {
		formats = append(formats, format)
	}
	return formats
}
