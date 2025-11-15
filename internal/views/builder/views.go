package builder

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	normalStyle   = lipgloss.NewStyle()
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	checkboxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

// renderWelcome renders the welcome screen
func (m builderModel) renderWelcome() string {
	var s strings.Builder

	s.WriteString("Welcome to the Interactive View Builder!\n\n")
	s.WriteString("This wizard will guide you through creating a custom view for displaying tasks.\n")
	s.WriteString(fmt.Sprintf("View name: %s\n\n", lipgloss.NewStyle().Bold(true).Render(m.builder.ViewName)))
	s.WriteString("You'll configure:\n")
	s.WriteString("  • View description\n")
	s.WriteString("  • Which fields to display\n")
	s.WriteString("  • Field order and formatting\n")
	s.WriteString("  • Display options\n\n")
	s.WriteString(dimStyle.Render("Press Enter to continue..."))

	return s.String()
}

// renderBasicInfo renders the basic info input screen
func (m builderModel) renderBasicInfo() string {
	var s strings.Builder

	s.WriteString("Basic Information\n\n")
	s.WriteString("Enter a description for your view (optional):\n\n")
	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("Press Enter to continue..."))

	return s.String()
}

// renderFieldSelection renders the field selection screen
func (m builderModel) renderFieldSelection() string {
	var s strings.Builder

	s.WriteString("Field Selection\n\n")
	s.WriteString("Select which fields to display in your view:\n\n")

	for i, field := range m.builder.AvailableFields {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		checkbox := "[ ]"
		if field.Selected {
			checkbox = checkboxStyle.Render("[✓]")
		}

		line := fmt.Sprintf("%s %s %s", cursor, checkbox, field.Name)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		s.WriteString(line)
		s.WriteString(dimStyle.Render(" - " + field.Description))
		s.WriteString("\n")
	}

	selectedCount := 0
	for _, field := range m.builder.AvailableFields {
		if field.Selected {
			selectedCount++
		}
	}

	s.WriteString(fmt.Sprintf("\n%s selected\n", dimStyle.Render(fmt.Sprintf("%d fields", selectedCount))))

	return s.String()
}

// renderFieldOrdering renders the field ordering screen
func (m builderModel) renderFieldOrdering() string {
	var s strings.Builder

	s.WriteString("Field Ordering\n\n")
	s.WriteString("Arrange the order fields will appear:\n\n")

	for i, fieldName := range m.builder.FieldOrder {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %d. %s", cursor, i+1, fieldName)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	s.WriteString(fmt.Sprintf("\n%s\n", dimStyle.Render("Use Ctrl+↑/↓ to move fields")))

	return s.String()
}

// renderFieldConfig renders the field configuration screen
func (m builderModel) renderFieldConfig() string {
	var s strings.Builder

	s.WriteString("Field Configuration\n\n")
	s.WriteString("Configure formatting for each field:\n\n")

	for i, fieldName := range m.builder.FieldOrder {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		// Find field details
		var field *FieldItem
		for j := range m.builder.AvailableFields {
			if m.builder.AvailableFields[j].Name == fieldName {
				field = &m.builder.AvailableFields[j]
				break
			}
		}

		if field == nil {
			continue
		}

		colorIndicator := ""
		if field.Color {
			colorIndicator = checkboxStyle.Render(" [color]")
		}

		line := fmt.Sprintf("%s %s: %s%s", cursor, fieldName, field.Format, colorIndicator)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	s.WriteString(fmt.Sprintf("\n%s\n", dimStyle.Render("Space to toggle color • Tab/Shift+Tab to navigate")))

	return s.String()
}

// renderDisplayOptions renders the display options screen
func (m builderModel) renderDisplayOptions() string {
	var s strings.Builder

	s.WriteString("Display Options\n\n")
	s.WriteString("Configure display settings:\n\n")

	options := []struct {
		name    string
		enabled bool
	}{
		{"Show header", m.builder.ShowHeader},
		{"Show border", m.builder.ShowBorder},
		{"Compact mode", m.builder.CompactMode},
	}

	for i, opt := range options {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}

		checkbox := "[ ]"
		if opt.enabled {
			checkbox = checkboxStyle.Render("[✓]")
		}

		line := fmt.Sprintf("%s %s %s", cursor, checkbox, opt.name)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}

		s.WriteString(line)
		s.WriteString("\n")
	}

	s.WriteString(fmt.Sprintf("\nDate format: %s\n", m.builder.DateFormat))

	return s.String()
}

// renderConfirm renders the confirmation screen
func (m builderModel) renderConfirm() string {
	var s strings.Builder

	s.WriteString("Confirm View Configuration\n\n")
	s.WriteString(fmt.Sprintf("Name: %s\n", m.builder.ViewName))
	s.WriteString(fmt.Sprintf("Description: %s\n", m.builder.ViewDescription))
	s.WriteString(fmt.Sprintf("\nFields (%d):\n", len(m.builder.FieldOrder)))

	for i, fieldName := range m.builder.FieldOrder {
		var field *FieldItem
		for j := range m.builder.AvailableFields {
			if m.builder.AvailableFields[j].Name == fieldName {
				field = &m.builder.AvailableFields[j]
				break
			}
		}

		if field != nil {
			colorIndicator := ""
			if field.Color {
				colorIndicator = " (color)"
			}
			s.WriteString(fmt.Sprintf("  %d. %s [%s]%s\n", i+1, field.Name, field.Format, colorIndicator))
		}
	}

	s.WriteString("\nDisplay Options:\n")
	s.WriteString(fmt.Sprintf("  Header: %v\n", m.builder.ShowHeader))
	s.WriteString(fmt.Sprintf("  Border: %v\n", m.builder.ShowBorder))
	s.WriteString(fmt.Sprintf("  Compact: %v\n", m.builder.CompactMode))
	s.WriteString(fmt.Sprintf("  Date format: %s\n", m.builder.DateFormat))
	if m.builder.SortBy != "" {
		s.WriteString(fmt.Sprintf("  Sort by: %s (%s)\n", m.builder.SortBy, m.builder.SortOrder))
	}

	if len(m.builder.FilterStatus) > 0 {
		s.WriteString("\nFilters:\n")
		s.WriteString(fmt.Sprintf("  Status: %v\n", m.builder.FilterStatus))
	}

	s.WriteString("\n")
	s.WriteString(selectedStyle.Render("Save this view? (y/n)"))

	return s.String()
}
