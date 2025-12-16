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
	result = strings.Replace(result, "{{PROXY_NAMES}}", proxyNames, -1)

	return result, nil
}

// generateClashProxies generates YAML for proxies section
func (r *TemplateRenderer) generateClashProxies(nodes []*Node, password string) (string, error) {
	proxies := make([]clashProxy, 0, len(nodes))

	for _, node := range nodes {
		var proxy clashProxy

		if node.Protocol == "trojan" {
			proxy = clashProxy{
				Name:           node.Name,
				Type:           "trojan",
				Server:         node.ServerAddress,
				Port:           node.SubscriptionPort,
				Password:       password,
				UDP:            true,
				SNI:            node.SNI,
				SkipCertVerify: node.AllowInsecure,
			}

			// Handle transport
			switch node.TransportProtocol {
			case "ws":
				proxy.Network = "ws"
				proxy.WSOpts = &clashWSOpts{
					Path: node.Path,
				}
				if node.Host != "" {
					proxy.WSOpts.Headers = map[string]string{
						"Host": node.Host,
					}
				}
			case "grpc":
				proxy.Network = "grpc"
				proxy.GRPCOpts = &clashGRPCOpts{
					GRPCServiceName: node.Host,
				}
			}
		} else {
			proxy = clashProxy{
				Name:     node.Name,
				Type:     "ss",
				Server:   node.ServerAddress,
				Port:     node.SubscriptionPort,
				Cipher:   node.EncryptionMethod,
				Password: password,
				UDP:      true,
			}

			if node.Plugin != "" {
				proxy.Plugin = node.Plugin
				proxy.PluginOpts = node.PluginOpts
			}
		}

		proxies = append(proxies, proxy)
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(proxies)
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

// extractProxyNames returns comma-separated node names
func (r *TemplateRenderer) extractProxyNames(nodes []*Node) string {
	if len(nodes) == 0 {
		return ""
	}

	names := make([]string, len(nodes))
	for i, node := range nodes {
		names[i] = node.Name
	}

	return strings.Join(names, ", ")
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
