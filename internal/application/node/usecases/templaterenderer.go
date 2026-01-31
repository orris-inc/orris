package usecases

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/orris-inc/orris/internal/infrastructure/template"
)

// TemplateRenderer renders subscription templates with node data
type TemplateRenderer struct {
	loader *template.SubscriptionTemplateLoader
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(loader *template.SubscriptionTemplateLoader) *TemplateRenderer {
	return &TemplateRenderer{
		loader: loader,
	}
}

// HasTemplate checks if a custom template exists for the given format
func (r *TemplateRenderer) HasTemplate(formatType string) bool {
	return r.loader.HasTemplate(formatType)
}

// RenderClash renders Clash template with node data
func (r *TemplateRenderer) RenderClash(nodes []*Node, password string) (string, error) {
	tmpl, ok := r.loader.Get("clash")
	if !ok {
		return "", fmt.Errorf("no clash template found")
	}

	// Generate proxies YAML
	proxiesYAML, err := r.generateClashProxies(nodes, password)
	if err != nil {
		return "", fmt.Errorf("failed to generate proxies YAML: %w", err)
	}

	// Generate proxy names list
	proxyNames := r.extractProxyNames(nodes)

	// Replace placeholders
	result := strings.Replace(tmpl, "# {{PROXIES}}", proxiesYAML, 1)
	result = strings.ReplaceAll(result, "{{PROXY_NAMES}}", proxyNames)

	return result, nil
}

// generateClashProxies generates YAML for proxies section
// Delegates to ClashFormatter for consistent proxy generation across all protocols
func (r *TemplateRenderer) generateClashProxies(nodes []*Node, password string) (string, error) {
	// Use ClashFormatter to generate proxies with full protocol support
	formatter := NewClashFormatter()
	content, err := formatter.FormatWithPassword(nodes, password)
	if err != nil {
		return "", fmt.Errorf("failed to generate proxies: %w", err)
	}

	// Parse the generated YAML to extract just the proxies array
	var config clashConfig
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return "", fmt.Errorf("failed to parse proxies YAML: %w", err)
	}

	// Marshal proxies back to YAML
	yamlBytes, err := yaml.Marshal(config.Proxies)
	if err != nil {
		return "", fmt.Errorf("failed to marshal proxies: %w", err)
	}

	// Indent each line with 2 spaces (to match Clash format under "proxies:")
	lines := strings.Split(strings.TrimSpace(string(yamlBytes)), "\n")
	indentedLines := make([]string, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			indentedLines[i] = "  " + line
		} else {
			indentedLines[i] = line
		}
	}

	return strings.Join(indentedLines, "\n"), nil
}

// extractProxyNames returns comma-separated node names with proper YAML escaping
func (r *TemplateRenderer) extractProxyNames(nodes []*Node) string {
	if len(nodes) == 0 {
		return ""
	}

	names := make([]string, len(nodes))
	for i, node := range nodes {
		names[i] = quoteYAMLString(node.Name)
	}

	return strings.Join(names, ", ")
}

// quoteYAMLString wraps node name in single quotes for YAML flow-style arrays.
// Single quotes in the name are escaped by doubling them (YAML spec).
func quoteYAMLString(s string) string {
	escaped := strings.ReplaceAll(s, "'", "''")
	return "'" + escaped + "'"
}

// RenderSurge renders Surge template with node data (placeholder for future implementation)
func (r *TemplateRenderer) RenderSurge(nodes []*Node, password string) (string, error) {
	tmpl, ok := r.loader.Get("surge")
	if !ok {
		return "", fmt.Errorf("no surge template found")
	}

	// TODO: Implement Surge template rendering
	// For now, just return the template as-is (no placeholder replacement)
	_ = nodes
	_ = password
	return tmpl, nil
}
