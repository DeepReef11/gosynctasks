package builder

import (
	"fmt"
	"gosynctasks/internal/views"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the interactive view builder in the terminal.
//
// This is the primary entry point for creating views through the interactive UI.
// It initializes a bubbletea program in alternate screen mode, guides the user
// through all configuration steps, and returns the built view.
//
// The builder guides users through:
//  1. View description (optional)
//  2. Field selection (must select at least one)
//  3. Field ordering (arrange display order)
//  4. Field configuration (formats, colors, widths)
//  5. Display options (headers, borders, sorting)
//  6. Confirmation (review and accept/cancel)
//
// User interaction:
//   - Arrow keys (↑/↓) navigate options
//   - Space toggles selections
//   - Ctrl+↑/↓ moves fields in ordering screen
//   - Tab/Shift+Tab navigate between fields in configuration
//   - Enter advances to next step
//   - Ctrl+C or Esc cancels at any time
//   - Y/N accepts or rejects at confirmation
//
// The viewName parameter becomes the view's identifier. It will be validated
// during the interactive flow (must be 1-50 chars, alphanumeric with _ and -).
//
// Returns the created view on success, or an error if the user cancels or
// validation fails.
//
// Example:
//
//	view, err := builder.Run("urgent-tasks")
//	if err != nil {
//	    log.Fatalf("Failed to create view: %v", err)
//	}
//	// Save the view to disk or use it...
func Run(viewName string) (*views.View, error) {
	builder := NewViewBuilder(viewName)
	model := newModel(builder)

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running interactive builder: %w", err)
	}

	// Get the result from the final model using type switch
	switch m := finalModel.(type) {
	case builderModel:
		if m.builder.CurrentState == StateCancelled {
			return nil, fmt.Errorf("cancelled by user")
		}

		if m.builder.View != nil {
			return m.builder.View, nil
		}

		// Build the view from builder state
		view, err := m.builder.BuildView()
		if err != nil {
			return nil, err
		}

		return view, nil
	default:
		return nil, fmt.Errorf("unexpected model type")
	}
}
