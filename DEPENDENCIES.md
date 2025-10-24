# Dependencies for Notification System

## Required Go Modules

The following Go modules are required for the notification and announcement system:

### Markdown Rendering
```bash
go get github.com/yuin/goldmark
```
- **Purpose**: Markdown parsing and rendering
- **Usage**: Convert markdown content in announcements and notifications to HTML
- **Version**: Latest stable

### HTML Sanitization
```bash
go get github.com/microcosm-cc/bluemonday
```
- **Purpose**: HTML sanitization and XSS protection
- **Usage**: Sanitize user-generated content in announcements to prevent XSS attacks
- **Version**: Latest stable

## Installation

Run the following commands to install all required dependencies:

```bash
go get github.com/yuin/goldmark
go get github.com/microcosm-cc/bluemonday
go mod tidy
```

## Usage Examples

### Markdown Rendering with Goldmark
```go
import (
    "bytes"
    "github.com/yuin/goldmark"
)

func RenderMarkdown(content string) (string, error) {
    var buf bytes.Buffer
    if err := goldmark.Convert([]byte(content), &buf); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

### HTML Sanitization with Bluemonday
```go
import (
    "github.com/microcosm-cc/bluemonday"
)

func SanitizeHTML(html string) string {
    p := bluemonday.UGCPolicy()
    return p.Sanitize(html)
}
```

## Notes
- All dependencies are already listed in go.mod after running `go get`
- Use `go mod tidy` to clean up unused dependencies
- These libraries are production-ready and widely used in the Go ecosystem
