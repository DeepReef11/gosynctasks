package builder

import (
	"gosynctasks/internal/views"
	"strings"
	"testing"
)

func TestValidateViewName(t *testing.T) {
	tests := []struct {
		name      string
		viewName  string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid name",
			viewName:  "my-view",
			shouldErr: false,
		},
		{
			name:      "valid with underscore",
			viewName:  "my_view",
			shouldErr: false,
		},
		{
			name:      "valid alphanumeric",
			viewName:  "view123",
			shouldErr: false,
		},
		{
			name:      "empty name",
			viewName:  "",
			shouldErr: true,
			errMsg:    "required",
		},
		{
			name:      "name too long",
			viewName:  strings.Repeat("a", 51),
			shouldErr: true,
			errMsg:    "between 1 and 50",
		},
		{
			name:      "invalid characters",
			viewName:  "my view",
			shouldErr: true,
			errMsg:    "letters, numbers, underscores, and hyphens",
		},
		{
			name:      "special characters",
			viewName:  "my@view",
			shouldErr: true,
			errMsg:    "letters, numbers, underscores, and hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewViewBuilder(tt.viewName)
			err := builder.ValidateViewName()

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestValidateFieldSelection(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*ViewBuilder)
		shouldErr bool
	}{
		{
			name: "default selection (status and summary)",
			setupFunc: func(b *ViewBuilder) {
				// Default builder has status and summary selected
			},
			shouldErr: false,
		},
		{
			name: "no fields selected",
			setupFunc: func(b *ViewBuilder) {
				// Deselect all fields
				for i := range b.AvailableFields {
					b.AvailableFields[i].Selected = false
				}
			},
			shouldErr: true,
		},
		{
			name: "one field selected",
			setupFunc: func(b *ViewBuilder) {
				// Deselect all
				for i := range b.AvailableFields {
					b.AvailableFields[i].Selected = false
				}
				// Select only one
				b.AvailableFields[0].Selected = true
			},
			shouldErr: false,
		},
		{
			name: "all fields selected",
			setupFunc: func(b *ViewBuilder) {
				// Select all fields
				for i := range b.AvailableFields {
					b.AvailableFields[i].Selected = true
				}
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewViewBuilder("test")
			tt.setupFunc(builder)

			err := builder.ValidateFieldSelection()

			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			} else if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if tt.shouldErr && err != nil {
				if !strings.Contains(err.Error(), "at least one field") {
					t.Errorf("Expected error about field selection, got %v", err)
				}
			}
		})
	}
}

func TestValidateFieldConfigs(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*ViewBuilder)
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid default configs",
			setupFunc: func(b *ViewBuilder) {
				// Default configs are all valid
			},
			shouldErr: false,
		},
		{
			name: "invalid format",
			setupFunc: func(b *ViewBuilder) {
				b.AvailableFields[0].Selected = true
				b.AvailableFields[0].Format = "invalid_format"
			},
			shouldErr: true,
			errMsg:    "invalid format",
		},
		{
			name: "width too large",
			setupFunc: func(b *ViewBuilder) {
				b.AvailableFields[0].Selected = true
				b.AvailableFields[0].Width = 300
			},
			shouldErr: true,
			errMsg:    "width must be between",
		},
		{
			name: "width negative",
			setupFunc: func(b *ViewBuilder) {
				b.AvailableFields[0].Selected = true
				b.AvailableFields[0].Width = -10
			},
			shouldErr: true,
			errMsg:    "width must be between",
		},
		{
			name: "valid custom width",
			setupFunc: func(b *ViewBuilder) {
				b.AvailableFields[0].Selected = true
				b.AvailableFields[0].Width = 50
			},
			shouldErr: false,
		},
		{
			name: "valid format change",
			setupFunc: func(b *ViewBuilder) {
				b.AvailableFields[1].Selected = true // summary field
				b.AvailableFields[1].Format = "truncate"
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewViewBuilder("test")
			tt.setupFunc(builder)

			err := builder.ValidateFieldConfigs()

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestBuildView_WithValidation(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*ViewBuilder)
		shouldErr bool
		errMsg    string
	}{
		{
			name: "valid view",
			setupFunc: func(b *ViewBuilder) {
				b.UpdateSelectedFields()
				b.UpdateFieldOrder()
			},
			shouldErr: false,
		},
		{
			name: "no fields selected",
			setupFunc: func(b *ViewBuilder) {
				for i := range b.AvailableFields {
					b.AvailableFields[i].Selected = false
				}
				b.UpdateSelectedFields()
				b.UpdateFieldOrder()
			},
			shouldErr: true,
			errMsg:    "at least one field",
		},
		{
			name: "invalid field format",
			setupFunc: func(b *ViewBuilder) {
				b.AvailableFields[0].Selected = true
				b.AvailableFields[0].Format = "invalid"
				b.UpdateSelectedFields()
				b.UpdateFieldOrder()
			},
			shouldErr: true,
			errMsg:    "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewViewBuilder("test")
			tt.setupFunc(builder)

			view, err := builder.BuildView()

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				if view != nil {
					t.Error("Expected nil view on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if view == nil {
					t.Error("Expected valid view, got nil")
				}
			}
		})
	}
}

func TestModelValidation_EmptyFieldSelection(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)

	// Deselect all fields
	for i := range m.builder.AvailableFields {
		m.builder.AvailableFields[i].Selected = false
	}

	m.builder.CurrentState = StateFieldSelection

	// Try to proceed with Enter
	updated, _ := m.handleEnter()
	m = updated.(builderModel)

	// Should stay at StateFieldSelection due to validation error
	if m.builder.CurrentState != StateFieldSelection {
		t.Error("Should stay at StateFieldSelection when validation fails")
	}

	// Should have error message
	if m.errorMsg == "" {
		t.Error("Expected error message when validation fails")
	}

	if !strings.Contains(m.errorMsg, "at least one field") {
		t.Errorf("Expected error about field selection, got %q", m.errorMsg)
	}
}

func TestModelValidation_ErrorClearing(t *testing.T) {
	builder := NewViewBuilder("test")
	m := newModel(builder)

	// Set an error message
	m.errorMsg = "test error"

	// Transition to next state (should clear error)
	m.builder.CurrentState = StateWelcome
	updated, _ := m.handleEnter()
	m = updated.(builderModel)

	// Error should be cleared
	if m.errorMsg != "" {
		t.Errorf("Expected error to be cleared, got %q", m.errorMsg)
	}
}

func TestViewsPackageValidation(t *testing.T) {
	// Test the centralized validation in views package

	t.Run("ValidateViewName", func(t *testing.T) {
		// Valid name
		if err := views.ValidateViewName("my-view"); err != nil {
			t.Errorf("Expected valid name, got error: %v", err)
		}

		// Invalid name
		if err := views.ValidateViewName(""); err == nil {
			t.Error("Expected error for empty name")
		}
	})

	t.Run("ValidateField", func(t *testing.T) {
		// Valid field
		field := views.FieldConfig{
			Name:   "status",
			Format: "symbol",
			Width:  50,
		}
		if err := views.ValidateField(&field); err != nil {
			t.Errorf("Expected valid field, got error: %v", err)
		}

		// Invalid field name
		invalidField := views.FieldConfig{
			Name: "nonexistent",
		}
		if err := views.ValidateField(&invalidField); err == nil {
			t.Error("Expected error for invalid field name")
		}

		// Invalid format
		invalidFormat := views.FieldConfig{
			Name:   "status",
			Format: "invalid",
		}
		if err := views.ValidateField(&invalidFormat); err == nil {
			t.Error("Expected error for invalid format")
		}
	})

	t.Run("ValidateView", func(t *testing.T) {
		// Valid view
		validView := &views.View{
			Name: "test",
			Fields: []views.FieldConfig{
				{Name: "status", Format: "symbol"},
				{Name: "summary", Format: "full"},
			},
			Display: views.DisplayOptions{
				SortBy:    "priority",
				SortOrder: "asc",
			},
		}
		if err := views.ValidateView(validView); err != nil {
			t.Errorf("Expected valid view, got error: %v", err)
		}

		// Invalid view (no fields)
		invalidView := &views.View{
			Name:   "test",
			Fields: []views.FieldConfig{},
		}
		if err := views.ValidateView(invalidView); err == nil {
			t.Error("Expected error for view with no fields")
		}
	})
}
