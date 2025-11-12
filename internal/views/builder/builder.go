package builder

import (
	"fmt"
	"gosynctasks/internal/views"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the interactive view builder and returns the built view or an error.
// The viewName parameter sets the name of the view being created.
// Returns an error if the user cancels or if validation fails.
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
