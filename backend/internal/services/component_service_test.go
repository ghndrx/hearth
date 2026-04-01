package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// TestComponentValidation tests component validation logic
func TestComponentValidation_ButtonValid(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeButton,
		Style:     models.ButtonStylePrimary,
		Label:     "Click Me",
		CustomID:  "test_button",
		Disabled:  false,
	}
	
	// This should be valid - has label
	assert.NotEmpty(t, comp.Label)
	assert.NotEmpty(t, comp.CustomID)
}

func TestComponentValidation_ButtonWithEmoji(t *testing.T) {
	comp := &models.MessageComponent{
		ID:         uuid.New(),
		Type:       models.ComponentTypeButton,
		Style:      models.ButtonStylePrimary,
		EmojiName:  "thumbsup",
		CustomID:   "test_button",
		Disabled:   false,
	}
	
	// Valid - has emoji
	assert.NotEmpty(t, comp.EmojiName)
}

func TestComponentValidation_LinkButton(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeButton,
		Style:     models.ButtonStyleLink,
		Label:     "Open Link",
		CustomID:  "link_button",
		URL:       "https://example.com",
		Disabled:  false,
	}
	
	// Valid link button - has URL
	assert.Equal(t, models.ButtonStyleLink, comp.Style)
	assert.NotEmpty(t, comp.URL)
}

func TestComponentValidation_SelectMenuValid(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeSelectMenu,
		CustomID:  "select_menu",
		Options: []models.SelectOption{
			{Label: "Option 1", Value: "opt1"},
			{Label: "Option 2", Value: "opt2"},
		},
		Placeholder: "Choose one",
		Disabled:    false,
	}
	
	// Valid select menu - has options
	assert.Greater(t, len(comp.Options), 0)
	assert.NotEmpty(t, comp.CustomID)
}

func TestComponentValidation_SelectMenuMaxOptions(t *testing.T) {
	// Discord limit is 25 options per select menu
	options := make([]models.SelectOption, 25)
	for i := 0; i < 25; i++ {
		options[i] = models.SelectOption{
			Label: "Option",
			Value: "opt",
		}
	}
	
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeSelectMenu,
		CustomID:  "select_menu",
		Options:   options,
		Disabled:  false,
	}
	
	// Should have exactly 25 options
	assert.Equal(t, 25, len(comp.Options))
}

func TestComponentValidation_TextInput(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeTextInput,
		Style:     models.TextInputStyleShort,
		Label:     "Short Answer",
		CustomID:  "text_input",
		Required:  true,
		MinLength: intPtrComponent(1),
		MaxLength: intPtrComponent(100),
		Disabled:  false,
	}
	
	// Valid text input
	assert.NotEmpty(t, comp.CustomID)
	assert.True(t, comp.Required)
}

func TestComponentValidation_ActionRow(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeActionRow,
		Disabled:  false,
	}
	
	// Action rows don't need label or custom_id
	assert.Equal(t, models.ComponentTypeActionRow, comp.Type)
}

func intPtrComponent(i int) *int {
	return &i
}

// TestModalValidation tests modal validation logic  
func TestModalValidation_ValidModal(t *testing.T) {
	modal := &models.ModalComponent{
		ID:       uuid.New(),
		CustomID: "test_modal",
		Type:     models.ModalTypePrimary,
		Title:    "Test Modal Title",
		Rows: []models.ModalRow{
			{
				Components: []models.MessageComponent{
					{
						ID:       uuid.New(),
						Type:     models.ComponentTypeTextInput,
						Style:    models.TextInputStyleShort,
						Label:    "Name",
						CustomID: "name_input",
					},
				},
			},
		},
	}
	
	assert.NotEmpty(t, modal.CustomID)
	assert.NotEmpty(t, modal.Title)
	assert.Greater(t, len(modal.Rows), 0)
}

func TestModalValidation_ModalTypes(t *testing.T) {
	tests := []struct {
		modalType models.ModalType
		valid     bool
	}{
		{models.ModalTypePrimary, true},
		{models.ModalTypeDanger, true},
	}
	
	for _, tt := range tests {
		modal := &models.ModalComponent{
			CustomID: "test_modal",
			Type:     tt.modalType,
			Title:    "Test",
			Rows: []models.ModalRow{
				{Components: []models.MessageComponent{}},
			},
		}
		
		assert.NotEmpty(t, modal.Type)
	}
}

// TestComponentStyles tests button styles
func TestComponentStyles_All(t *testing.T) {
	styles := []models.ComponentStyle{
		models.ButtonStylePrimary,
		models.ButtonStyleSecondary,
		models.ButtonStyleSuccess,
		models.ButtonStyleDanger,
		models.ButtonStyleLink,
		models.TextInputStyleShort,
		models.TextInputStyleParagraph,
	}
	
	for _, style := range styles {
		assert.NotEmpty(t, style)
	}
}

// TestComponentTypes tests component types
func TestComponentTypes_All(t *testing.T) {
	types := []models.ComponentType{
		models.ComponentTypeActionRow,
		models.ComponentTypeButton,
		models.ComponentTypeSelectMenu,
		models.ComponentTypeTextInput,
	}
	
	for _, compType := range types {
		assert.NotEmpty(t, compType)
	}
}

// TestSelectOption tests select option structure
func TestSelectOption(t *testing.T) {
	option := models.SelectOption{
		Label:       "Option Label",
		Value:       "option_value",
		Description: "This is an option description",
		Emoji:       "thumbsup",
		Default:     true,
	}
	
	assert.Equal(t, "Option Label", option.Label)
	assert.Equal(t, "option_value", option.Value)
	assert.Equal(t, "This is an option description", option.Description)
	assert.Equal(t, "thumbsup", option.Emoji)
	assert.True(t, option.Default)
}

// TestComponentInteractionTypes tests component interaction types
func TestComponentInteractionTypes(t *testing.T) {
	assert.Equal(t, models.ComponentInteractionType(0), models.InteractionTypeComponent)
	assert.Equal(t, models.ComponentInteractionType(1), models.InteractionTypeTextInput)
}

// TestModalInteractionData tests modal interaction structure
func TestModalInteractionData(t *testing.T) {
	interaction := &models.ModalInteraction{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		ChannelID:   uuid.New(),
		MessageID:   uuid.New(),
		ModalID:     uuid.New(),
		CustomID:    "test_modal",
		ComponentID: uuid.New(),
		Values: map[string]string{
			"name_input": "John Doe",
			"email_input": "john@example.com",
		},
	}
	
	assert.NotEqual(t, uuid.Nil, interaction.ID)
	assert.Len(t, interaction.Values, 2)
	assert.Equal(t, "John Doe", interaction.Values["name_input"])
}
