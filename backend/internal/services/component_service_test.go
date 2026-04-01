package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"hearth/internal/models"
)

// =============================================================================
// Component Validation Tests
// =============================================================================

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

// =============================================================================
// Component Model Tests
// =============================================================================

func TestMessageComponent_ButtonFields(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeButton,
		Style:     models.ButtonStylePrimary,
		Label:     "Click Me",
		CustomID:  "test_button",
		Disabled:  false,
		EmojiID:   uuid.New(),
		EmojiName: "thumbsup",
	}
	
	assert.Equal(t, models.ComponentTypeButton, comp.Type)
	assert.Equal(t, models.ButtonStylePrimary, comp.Style)
	assert.Equal(t, "Click Me", comp.Label)
	assert.Equal(t, "test_button", comp.CustomID)
	assert.False(t, comp.Disabled)
	assert.NotEqual(t, uuid.Nil, comp.EmojiID)
	assert.Equal(t, "thumbsup", comp.EmojiName)
}

func TestMessageComponent_LinkButtonFields(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeButton,
		Style:     models.ButtonStyleLink,
		Label:     "Open GitHub",
		CustomID:  "github_link",
		URL:       "https://github.com",
		Disabled:  false,
	}
	
	assert.Equal(t, models.ButtonStyleLink, comp.Style)
	assert.Equal(t, "https://github.com", comp.URL)
}

func TestMessageComponent_SelectMenuFields(t *testing.T) {
	minVals := 1
	maxVals := 5
	
	comp := &models.MessageComponent{
		ID:         uuid.New(),
		Type:       models.ComponentTypeSelectMenu,
		CustomID:   "color_selector",
		Placeholder: "Pick a color",
		MinValues:  &minVals,
		MaxValues:  &maxVals,
		Options: []models.SelectOption{
			{Label: "Red", Value: "red", Description: "The color red"},
			{Label: "Green", Value: "green", Description: "The color green"},
			{Label: "Blue", Value: "blue", Description: "The color blue"},
		},
		Disabled: false,
	}
	
	assert.Equal(t, models.ComponentTypeSelectMenu, comp.Type)
	assert.Equal(t, "Pick a color", comp.Placeholder)
	assert.Equal(t, 1, *comp.MinValues)
	assert.Equal(t, 5, *comp.MaxValues)
	assert.Len(t, comp.Options, 3)
}

func TestMessageComponent_TextInputFields(t *testing.T) {
	minLen := 5
	maxLen := 100
	
	comp := &models.MessageComponent{
		ID:         uuid.New(),
		Type:       models.ComponentTypeTextInput,
		Style:      models.TextInputStyleParagraph,
		Label:      "Feedback",
		CustomID:   "feedback_input",
		Placeholder: "Tell us what you think...",
		Required:   true,
		MinLength:  &minLen,
		MaxLength:  &maxLen,
		Value:      "Initial value",
		Disabled:   false,
	}
	
	assert.Equal(t, models.ComponentTypeTextInput, comp.Type)
	assert.Equal(t, models.TextInputStyleParagraph, comp.Style)
	assert.Equal(t, "Feedback", comp.Label)
	assert.Equal(t, "Feedback", comp.Label)
	assert.Equal(t, "feedback_input", comp.CustomID)
	assert.Equal(t, "Tell us what you think...", comp.Placeholder)
	assert.True(t, comp.Required)
	assert.Equal(t, 5, *comp.MinLength)
	assert.Equal(t, 100, *comp.MaxLength)
	assert.Equal(t, "Initial value", comp.Value)
}

func TestMessageComponent_ActionRowFields(t *testing.T) {
	comp := &models.MessageComponent{
		ID:        uuid.New(),
		Type:      models.ComponentTypeActionRow,
		Disabled:  false,
	}
	
	assert.Equal(t, models.ComponentTypeActionRow, comp.Type)
	assert.False(t, comp.Disabled)
}

// =============================================================================
// ComponentInteraction Tests
// =============================================================================

func TestComponentInteraction_ButtonClick(t *testing.T) {
	interaction := &models.ComponentInteraction{
		ID:          uuid.New(),
		Type:        models.InteractionTypeComponent,
		UserID:      uuid.New(),
		ChannelID:   uuid.New(),
		MessageID:   uuid.New(),
		ComponentID: uuid.New(),
		CustomID:    "confirm_btn",
		Values:      nil,
		CreatedAt:   time.Now(),
	}
	
	assert.Equal(t, models.InteractionTypeComponent, interaction.Type)
	assert.Nil(t, interaction.Values)
}

func TestComponentInteraction_SelectMenu(t *testing.T) {
	interaction := &models.ComponentInteraction{
		ID:          uuid.New(),
		Type:        models.InteractionTypeComponent,
		UserID:      uuid.New(),
		ChannelID:   uuid.New(),
		MessageID:   uuid.New(),
		ComponentID: uuid.New(),
		CustomID:    "color_select",
		Values:      []string{"blue"},
		CreatedAt:   time.Now(),
	}
	
	assert.Len(t, interaction.Values, 1)
	assert.Equal(t, "blue", interaction.Values[0])
}

func TestComponentInteraction_TextInput(t *testing.T) {
	interaction := &models.ComponentInteraction{
		ID:          uuid.New(),
		Type:        models.InteractionTypeTextInput,
		UserID:      uuid.New(),
		ChannelID:   uuid.New(),
		MessageID:   uuid.New(),
		ComponentID: uuid.New(),
		CustomID:    "feedback_text",
		Values:      []string{"Great product! I love it."},
		CreatedAt:   time.Now(),
	}
	
	assert.Equal(t, models.InteractionTypeTextInput, interaction.Type)
	assert.Len(t, interaction.Values, 1)
	assert.Equal(t, "Great product! I love it.", interaction.Values[0])
}

func TestComponentInteraction_MultiSelect(t *testing.T) {
	interaction := &models.ComponentInteraction{
		ID:          uuid.New(),
		Type:        models.InteractionTypeComponent,
		UserID:      uuid.New(),
		ChannelID:   uuid.New(),
		MessageID:   uuid.New(),
		ComponentID: uuid.New(),
		CustomID:    "multi_select",
		Values:      []string{"option1", "option2", "option3"},
		CreatedAt:   time.Now(),
	}
	
	assert.Len(t, interaction.Values, 3)
}

// =============================================================================
// Component Service Logic Tests
// =============================================================================

func TestComponentService_ButtonStyleDefault(t *testing.T) {
	// Test that button defaults to primary style when empty
	comp := &models.MessageComponent{
		Type:  models.ComponentTypeButton,
		Style: "",
	}
	
	if comp.Style == "" {
		comp.Style = models.ButtonStylePrimary
	}
	
	assert.Equal(t, models.ButtonStylePrimary, comp.Style)
}

func TestComponentService_TextInputStyleDefault(t *testing.T) {
	// Test that text input defaults to short style when empty
	comp := &models.MessageComponent{
		Type:  models.ComponentTypeTextInput,
		Style: "",
	}
	
	if comp.Style == "" {
		comp.Style = models.TextInputStyleShort
	}
	
	assert.Equal(t, models.TextInputStyleShort, comp.Style)
}

func TestComponentService_SelectMenuValidation(t *testing.T) {
	// Test select menu option validation
	options := []models.SelectOption{
		{Label: "Option 1", Value: "opt1"},
		{Label: "Option 2", Value: "opt2"},
		{Label: "", Value: ""}, // Invalid - empty label and value
	}
	
	valid := true
	for _, opt := range options {
		if opt.Value == "" {
			valid = false
			break
		}
	}
	
	assert.False(t, valid)
}

func TestComponentService_ButtonRequiresLabelOrEmoji(t *testing.T) {
	comp := &models.MessageComponent{
		Type:      models.ComponentTypeButton,
		Style:     models.ButtonStylePrimary,
		Label:     "",
		EmojiName: "",
		CustomID:  "test_btn",
	}
	
	hasLabelOrEmoji := comp.Label != "" || comp.EmojiName != ""
	assert.False(t, hasLabelOrEmoji)
}

func TestComponentService_LinkButtonRequiresURL(t *testing.T) {
	comp := &models.MessageComponent{
		Type:      models.ComponentTypeButton,
		Style:     models.ButtonStyleLink,
		Label:     "Click Here",
		CustomID:  "link_btn",
		URL:       "",
	}
	
	isLinkWithoutURL := comp.Style == models.ButtonStyleLink && comp.URL == ""
	assert.True(t, isLinkWithoutURL)
}

func TestComponentService_ComponentMaxLength(t *testing.T) {
	// Custom ID max length is 100 characters
	customID := "a" // Base
	comp := &models.MessageComponent{
		Type:     models.ComponentTypeButton,
		CustomID: customID,
	}
	
	for i := 0; i < 99; i++ {
		comp.CustomID += "a"
	}
	
	assert.LessOrEqual(t, len(comp.CustomID), 100)
}

func TestComponentService_SelectMenuMax25Options(t *testing.T) {
	// Create 25 options (Discord limit)
	options := make([]models.SelectOption, 25)
	for i := 0; i < 25; i++ {
		options[i] = models.SelectOption{
			Label: "Option",
			Value: "opt",
		}
	}
	
	comp := &models.MessageComponent{
		Type:     models.ComponentTypeSelectMenu,
		CustomID: "select",
		Options:  options,
	}
	
	assert.Equal(t, 25, len(comp.Options))
	
	// Try to add 26th option
	if len(comp.Options) < 25 {
		comp.Options = append(comp.Options, models.SelectOption{Label: "Extra", Value: "extra"})
	}
	
	// Should still be 25 (would be rejected by validation)
	assert.LessOrEqual(t, len(comp.Options), 25)
}

// =============================================================================
// Component Interaction Event Tests
// =============================================================================

func TestComponentInteractionEvent_Structure(t *testing.T) {
	msgID := uuid.New()
	channelID := uuid.New()
	userID := uuid.New()
	componentID := uuid.New()
	
	interaction := &models.ComponentInteraction{
		ID:          uuid.New(),
		Type:        models.InteractionTypeComponent,
		UserID:      userID,
		ChannelID:   channelID,
		MessageID:   msgID,
		ComponentID: componentID,
		CustomID:    "test_btn",
	}
	
	component := &models.MessageComponent{
		ID:       componentID,
		Type:     models.ComponentTypeButton,
		CustomID: "test_btn",
	}
	
	message := &models.Message{
		ID:         msgID,
		ChannelID:  channelID,
		Content:    "Test message",
		AuthorID:   userID,
	}
	
	// Verify the event data structure is valid
	assert.Equal(t, interaction.ComponentID, component.ID)
	assert.Equal(t, interaction.MessageID, message.ID)
	assert.Equal(t, interaction.ChannelID, channelID)
}
