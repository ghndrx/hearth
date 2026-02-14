package services

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"strings"
)

// SvelteService defines the interface for rendering Svelte HTML.
// Using an interface allows for mocking or swapping implementations
// (e.g., rendering to HTML vs. a custom template engine).
type SvelteService interface {
	// RenderConnectionStatus generates the HTML for a user's connection status.
	// It accepts a user ID to simulate dynamic data injection.
	RenderConnectionStatus(userID string) (string, error)
	
	// GenerateChatView renders the main conversation container.
	// Typically involves loading components for Messages and Input.
	GenerateChatView() (string, error)
}

// DefaultSvelteService implements SvelteService using Go's text/template
// to simulate the partials and markup structure expected from a build step.
type DefaultSvelteService struct {
	// We embed a methods struct or keep handlers here. 
	// For this example, we use a simple implementation pattern.
}

// NewSvelteService constructs a new instance of the service.
func NewSvelteService() SvelteService {
	return &DefaultSvelteService{}
}

// RenderConnectionStatus implements the interface.
// In a real-world scenario, this would call a command line tool (svelte-kit) 
// or an embedded binary function. Here, we use Go templates for the mockup.
func (s *DefaultSvelteService) RenderConnectionStatus(userID string) (string, error) {
	
	// Template to simulate a Svelte Component with data props
	// {#if active} is typical Svelte syntax we mimic
	const tpl = `<!-- Svelte Component: ConnectionStatus.svelte -->
<div class="connection-status" data-user-id="{{ .UserID }}">
	<span class="status-icon">
		{#{if .Active} 
			✅ Online
		{:else if .Away} 
			💛 Away
 		{/if} #}
	</span>
	<span class="details">
		<span class="status-dot {{ if .Active }}green{{ else }}gray{{ end }}"></span>
		{{ .UserID }}
	</span>
</div>`

	data := struct {
		UserID string
		Active bool
		Away   bool
	}{
		UserID: userID,
		Active: true, // Simulated dynamic state
		Away:   false,
	}

	var buf bytes.Buffer
	t := template.Must(template.New("svelte_status").Parse(tpl))
	err := t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute connection status template: %w", err)
	}

	return buf.String(), nil
}

// GenerateChatView implements the interface.
func (s *DefaultSvelteService) GenerateChatView() (string, error) {
	const chatTpl = `<!-- Svelte Component: ChatView.svelte -->
<div class="chat-container">
	<div class="sidebar">
		<!-- Sidebar Component -->
	</div>
	
	<main class="chat-main">
		<div class="server-list">
			{#each messages as msg (msg.id)}
				<div class="message {{ if .isSystem }}system{{ else }}user{{ end }}">
					<strong>{{ .author }}</strong>: {{ .content }}
				</div>
			{/each}
		</div>
		
		<div class="input-area">
			<input type="text" placeholder="Message #general" />
		</div>
	</main>
</div>`

	// In a real app, this might fetch data from a MessageService
	messages := []struct {
		author string
		content string
		id     int
	}{
		{"User1", "Hello from Discord!", 1},
		{"User2", "How's the Go service running?", 2},
	}

	data := struct {
		Messages []struct{ author string; content string; id int }
	}{
		Messages: messages,
	}

	var buf bytes.Buffer
	t := template.Must(template.New("svelte_chat").Parse(chatTpl))
	err := t.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute chat view template: %w", err)
	}

	return buf.String(), nil
}

// MockRenderer is a specialized struct for testing, 
// allowing us to capture the exact string content generated.
type MockRenderer struct {
	SvelteService
}

func (m *MockRenderer) CaptureOutput(callback func(SvelteService)) string {
	// This is a helper pattern; typically refactoring the service
	// to accept an io.Writer would be cleaner, but this demonstrates isolation.
	panic("utilizing a simpler approach for this unit test definition")
}