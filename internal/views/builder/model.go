package builder

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// builderModel is the bubbletea builderModel for the view builder
type builderModel struct {
	builder       *ViewBuilder
	textInput     textinput.Model
	cursor        int
	selectedIndex int
	quitting      bool
	width         int
	height        int
}

// newModel creates a new bubbletea builderModel
func newModel(builder *ViewBuilder) builderModel {
	ti := textinput.New()
	ti.Placeholder = "Enter view description..."
	ti.Focus()
	ti.Width = 50

	return builderModel{
		builder:   builder,
		textInput: ti,
		cursor:    0,
		width:     80,
		height:    24,
	}
}

// Init initializes the builderModel
func (m builderModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates builderModel state
func (m builderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.builder.CurrentState = StateCancelled
			m.quitting = true
			return m, tea.Quit

		case "y", "Y":
			if m.builder.CurrentState == StateConfirm {
				// Build and save
				view, err := m.builder.BuildView()
				if err != nil {
					m.builder.Err = err
					m.builder.CurrentState = StateCancelled
				} else {
					m.builder.View = view
					m.builder.CurrentState = StateDone
				}
				m.quitting = true
				return m, tea.Quit
			}

		case "n", "N":
			if m.builder.CurrentState == StateConfirm {
				m.builder.CurrentState = StateCancelled
				m.quitting = true
				return m, tea.Quit
			}

		case "enter":
			return m.handleEnter()

		case "up", "k":
			return m.handleUp()

		case "down", "j":
			return m.handleDown()

		case " ":
			return m.handleSpace()

		case "ctrl+up":
			return m.handleMoveUp()

		case "ctrl+down":
			return m.handleMoveDown()

		case "tab":
			return m.handleNext()

		case "shift+tab":
			return m.handlePrevious()
		}
	}

	// Handle text input for certain states
	if m.builder.CurrentState == StateBasicInfo {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the UI
func (m builderModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	// Header
	s.WriteString(m.renderHeader())
	s.WriteString("\n\n")

	// Current state view
	switch m.builder.CurrentState {
	case StateWelcome:
		s.WriteString(m.renderWelcome())
	case StateBasicInfo:
		s.WriteString(m.renderBasicInfo())
	case StateFieldSelection:
		s.WriteString(m.renderFieldSelection())
	case StateFieldOrdering:
		s.WriteString(m.renderFieldOrdering())
	case StateFieldConfig:
		s.WriteString(m.renderFieldConfig())
	case StateDisplayOptions:
		s.WriteString(m.renderDisplayOptions())
	case StateConfirm:
		s.WriteString(m.renderConfirm())
	}

	s.WriteString("\n\n")
	s.WriteString(m.renderHelp())

	return s.String()
}

// renderHeader renders the header with current state
func (m builderModel) renderHeader() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("Interactive View Builder")

	step := fmt.Sprintf("Step %d/6: %s", m.getStateNumber(), m.builder.CurrentState.String())
	stepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	return fmt.Sprintf("%s\n%s", title, stepStyle.Render(step))
}

// getStateNumber returns the current step number
func (m builderModel) getStateNumber() int {
	switch m.builder.CurrentState {
	case StateWelcome:
		return 0
	case StateBasicInfo:
		return 1
	case StateFieldSelection:
		return 2
	case StateFieldOrdering:
		return 3
	case StateFieldConfig:
		return 4
	case StateDisplayOptions:
		return 5
	case StateConfirm:
		return 6
	default:
		return 0
	}
}

// renderHelp renders the help text at the bottom
func (m builderModel) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	var help string
	switch m.builder.CurrentState {
	case StateWelcome:
		help = "enter: continue • ctrl+c: cancel"
	case StateBasicInfo:
		help = "enter: continue • ctrl+c: cancel"
	case StateFieldSelection:
		help = "↑/↓: navigate • space: toggle • enter: continue • ctrl+c: cancel"
	case StateFieldOrdering:
		help = "↑/↓: navigate • ctrl+↑/↓: move • enter: continue • ctrl+c: cancel"
	case StateFieldConfig:
		help = "↑/↓: navigate • space: toggle • tab: next field • enter: continue • ctrl+c: cancel"
	case StateDisplayOptions:
		help = "↑/↓: navigate • space: toggle • enter: continue • ctrl+c: cancel"
	case StateConfirm:
		help = "y: save • n: cancel • ctrl+c: cancel"
	}

	return helpStyle.Render(help)
}

// State transition handlers

func (m builderModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.builder.CurrentState {
	case StateWelcome:
		m.builder.CurrentState = StateBasicInfo
		m.textInput.SetValue(m.builder.ViewDescription)
		m.textInput.Focus()

	case StateBasicInfo:
		m.builder.ViewDescription = m.textInput.Value()
		m.builder.CurrentState = StateFieldSelection
		m.cursor = 0

	case StateFieldSelection:
		m.builder.UpdateSelectedFields()
		m.builder.UpdateFieldOrder()
		m.builder.CurrentState = StateFieldOrdering
		m.cursor = 0

	case StateFieldOrdering:
		m.builder.CurrentState = StateFieldConfig
		m.cursor = 0
		m.selectedIndex = 0

	case StateFieldConfig:
		m.builder.CurrentState = StateDisplayOptions
		m.cursor = 0

	case StateDisplayOptions:
		m.builder.CurrentState = StateConfirm
		m.cursor = 0

	case StateConfirm:
		// Build and save
		view, err := m.builder.BuildView()
		if err != nil {
			m.builder.Err = err
			m.builder.CurrentState = StateCancelled
		} else {
			m.builder.View = view
			m.builder.CurrentState = StateDone
		}
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m builderModel) handleUp() (tea.Model, tea.Cmd) {
	if m.cursor > 0 {
		m.cursor--
	}
	return m, nil
}

func (m builderModel) handleDown() (tea.Model, tea.Cmd) {
	maxCursor := m.getMaxCursor()
	if m.cursor < maxCursor {
		m.cursor++
	}
	return m, nil
}

func (m builderModel) handleSpace() (tea.Model, tea.Cmd) {
	switch m.builder.CurrentState {
	case StateFieldSelection:
		if m.cursor < len(m.builder.AvailableFields) {
			m.builder.AvailableFields[m.cursor].Selected = !m.builder.AvailableFields[m.cursor].Selected
		}

	case StateFieldConfig:
		// Toggle color for current field
		if m.cursor < len(m.builder.FieldOrder) {
			fieldName := m.builder.FieldOrder[m.cursor]
			for i := range m.builder.AvailableFields {
				if m.builder.AvailableFields[i].Name == fieldName {
					m.builder.AvailableFields[i].Color = !m.builder.AvailableFields[i].Color
					break
				}
			}
		}

	case StateDisplayOptions:
		switch m.cursor {
		case 0:
			m.builder.ShowHeader = !m.builder.ShowHeader
		case 1:
			m.builder.ShowBorder = !m.builder.ShowBorder
		case 2:
			m.builder.CompactMode = !m.builder.CompactMode
		}
	}

	return m, nil
}

func (m builderModel) handleMoveUp() (tea.Model, tea.Cmd) {
	if m.builder.CurrentState == StateFieldOrdering && m.cursor > 0 {
		// Swap with previous
		m.builder.FieldOrder[m.cursor], m.builder.FieldOrder[m.cursor-1] =
			m.builder.FieldOrder[m.cursor-1], m.builder.FieldOrder[m.cursor]
		m.cursor--
	}
	return m, nil
}

func (m builderModel) handleMoveDown() (tea.Model, tea.Cmd) {
	if m.builder.CurrentState == StateFieldOrdering && m.cursor < len(m.builder.FieldOrder)-1 {
		// Swap with next
		m.builder.FieldOrder[m.cursor], m.builder.FieldOrder[m.cursor+1] =
			m.builder.FieldOrder[m.cursor+1], m.builder.FieldOrder[m.cursor]
		m.cursor++
	}
	return m, nil
}

func (m builderModel) handleNext() (tea.Model, tea.Cmd) {
	if m.builder.CurrentState == StateFieldConfig {
		if m.cursor < len(m.builder.FieldOrder)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m builderModel) handlePrevious() (tea.Model, tea.Cmd) {
	if m.builder.CurrentState == StateFieldConfig {
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return m, nil
}

func (m builderModel) getMaxCursor() int {
	switch m.builder.CurrentState {
	case StateFieldSelection:
		return len(m.builder.AvailableFields) - 1
	case StateFieldOrdering:
		return len(m.builder.FieldOrder) - 1
	case StateFieldConfig:
		return len(m.builder.FieldOrder) - 1
	case StateDisplayOptions:
		return 2 // 3 options: ShowHeader, ShowBorder, CompactMode
	default:
		return 0
	}
}
