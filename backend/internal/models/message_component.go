package models

import (
	"time"

	"github.com/google/uuid"
)

// ComponentType represents the type of message component
type ComponentType string

const (
	ComponentTypeActionRow  ComponentType = "action_row"
	ComponentTypeButton     ComponentType = "button"
	ComponentTypeSelectMenu ComponentType = "select_menu"
	ComponentTypeTextInput  ComponentType = "text_input"
)

// ComponentStyle represents the visual style of a component
type ComponentStyle string

const (
	ButtonStylePrimary      ComponentStyle = "primary"
	ButtonStyleSecondary    ComponentStyle = "secondary"
	ButtonStyleSuccess      ComponentStyle = "success"
	ButtonStyleDanger       ComponentStyle = "danger"
	ButtonStyleLink         ComponentStyle = "link"
	TextInputStyleShort     ComponentStyle = "short"
	TextInputStyleParagraph ComponentStyle = "paragraph"
)

// SelectOption represents an option in a select menu
type SelectOption struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Emoji       string `json:"emoji,omitempty"`
	Default     bool   `json:"default,omitempty"`
}

// MessageEmoji represents an emoji on a component
type MessageEmoji struct {
	ID   uuid.UUID `json:"id,omitempty"`
	Name string    `json:"name,omitempty"`
	Anim bool      `json:"animated,omitempty"`
}

// MessageComponent represents an interactive component on a message
type MessageComponent struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	MessageID uuid.UUID      `json:"message_id" db:"message_id"`
	Type      ComponentType  `json:"type" db:"type"`
	Style     ComponentStyle `json:"style,omitempty" db:"style"`
	Label     string         `json:"label,omitempty" db:"label"`
	CustomID  string         `json:"custom_id" db:"custom_id"`
	URL       string         `json:"url,omitempty" db:"url"`
	Disabled  bool           `json:"disabled" db:"disabled"`
	EmojiID   uuid.UUID      `json:"emoji_id,omitempty" db:"emoji_id"`
	EmojiName string         `json:"emoji_name,omitempty" db:"emoji_name"`

	// Select menu specific
	Options     []SelectOption `json:"options,omitempty" db:"options"`
	MinValues   *int           `json:"min_values,omitempty" db:"min_values"`
	MaxValues   *int           `json:"max_values,omitempty" db:"max_values"`
	Placeholder string         `json:"placeholder,omitempty" db:"placeholder"`

	// Text input specific
	Required  bool   `json:"required,omitempty" db:"required"`
	Value     string `json:"value,omitempty" db:"value"`
	MinLength *int   `json:"min_length,omitempty" db:"min_length"`
	MaxLength *int   `json:"max_length,omitempty" db:"max_length"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ComponentInteractionType represents the type of component interaction
type ComponentInteractionType int

const (
	InteractionTypeComponent ComponentInteractionType = iota
	InteractionTypeTextInput
)

// ComponentInteraction represents a user's interaction with a component
type ComponentInteraction struct {
	ID          uuid.UUID                `json:"id" db:"id"`
	Type        ComponentInteractionType `json:"type" db:"type"`
	UserID      uuid.UUID                `json:"user_id" db:"user_id"`
	ChannelID   uuid.UUID                `json:"channel_id" db:"channel_id"`
	MessageID   uuid.UUID                `json:"message_id" db:"message_id"`
	ComponentID uuid.UUID                `json:"component_id" db:"component_id"`
	CustomID    string                   `json:"custom_id" db:"custom_id"`
	Values      []string                 `json:"values,omitempty" db:"values"`
	CreatedAt   time.Time                `json:"created_at" db:"created_at"`
}

// CreateComponentRequest is the request to create a component
type CreateComponentRequest struct {
	Type        ComponentType  `json:"type"`
	Style       ComponentStyle `json:"style,omitempty"`
	Label       string         `json:"label,omitempty"`
	CustomID    string         `json:"custom_id"`
	URL         string         `json:"url,omitempty"`
	Disabled    bool           `json:"disabled,omitempty"`
	Emoji       string         `json:"emoji,omitempty"`
	Options     []SelectOption `json:"options,omitempty"`
	MinValues   *int           `json:"min_values,omitempty"`
	MaxValues   *int           `json:"max_values,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Value       string         `json:"value,omitempty"`
	MinLength   *int           `json:"min_length,omitempty"`
	MaxLength   *int           `json:"max_length,omitempty"`
}

// UpdateComponentsRequest is the request to update all components on a message
type UpdateComponentsRequest struct {
	Components []CreateComponentRequest `json:"components"`
}

// HandleInteractionRequest is the request to handle a component interaction
type HandleInteractionRequest struct {
	CustomID string   `json:"custom_id"`
	Values   []string `json:"values,omitempty"`
}

// ModalType represents the type of modal
type ModalType string

const (
	ModalTypePrimary ModalType = "primary"
	ModalTypeDanger  ModalType = "danger"
)

// ModalComponent represents a modal triggered by a component interaction
type ModalComponent struct {
	ID       uuid.UUID      `json:"id" db:"id"`
	CustomID string         `json:"custom_id" db:"custom_id"`
	Type     ModalType      `json:"type" db:"type"`
	Title    string         `json:"title" db:"title"`
	Rows     []ModalRow     `json:"rows"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
}

// ModalRow represents a row in a modal containing components
type ModalRow struct {
	Components []MessageComponent `json:"components"`
}

// ModalInteraction represents a user's interaction with a modal
type ModalInteraction struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	UserID       uuid.UUID         `json:"user_id" db:"user_id"`
	ChannelID    uuid.UUID         `json:"channel_id" db:"channel_id"`
	MessageID    uuid.UUID         `json:"message_id" db:"message_id"`
	ModalID      uuid.UUID         `json:"modal_id" db:"modal_id"`
	CustomID     string            `json:"custom_id" db:"custom_id"`
	ComponentID  uuid.UUID         `json:"component_id" db:"component_id"`
	Values       map[string]string `json:"values" db:"values"`
	SubmittedAt  time.Time         `json:"submitted_at" db:"submitted_at"`
}

// CreateModalRequest is the request to create a modal
type CreateModalRequest struct {
	CustomID string                `json:"custom_id"`
	Type     ModalType            `json:"type"`
	Title    string               `json:"title"`
	Rows     []CreateModalRowRequest `json:"rows"`
}

// CreateModalRowRequest is the request to create a modal row
type CreateModalRowRequest struct {
	Components []CreateComponentRequest `json:"components"`
}

// ModalSubmitRequest is the request to submit a modal
type ModalSubmitRequest struct {
	MessageID   string            `json:"message_id"`
	ChannelID   string            `json:"channel_id"`
	ModalID     string            `json:"modal_id"`
	CustomID    string            `json:"custom_id"`
	ComponentID string            `json:"component_id"`
	Values      map[string]string `json:"values"`
}

// ComponentInteractionResponse is sent back to the client after a component interaction
type ComponentInteractionResponse struct {
	Type    InteractionResponseType `json:"type"`
	Content *InteractionContent    `json:"content,omitempty"`
	Data    *InteractionData       `json:"data,omitempty"`
}

// InteractionResponseType defines the type of response
type InteractionResponseType string

const (
	InteractionResponseAcknowledge InteractionResponseType = "acknowledge"
	InteractionResponseUpdateMessage InteractionResponseType = "update_message"
	InteractionResponseShowModal    InteractionResponseType = "show_modal"
	InteractionResponseError        InteractionResponseType = "error"
)

// InteractionContent is the content shown after interaction
type InteractionContent struct {
	Message   *Message `json:"message,omitempty"`
	Modal     *ModalComponent `json:"modal,omitempty"`
	Ephemeral bool   `json:"ephemeral,omitempty"`
}

// InteractionData is additional data for the interaction response
type InteractionData struct {
	URL         string `json:"url,omitempty"`
	SelectType  string `json:"select_type,omitempty"`
}
