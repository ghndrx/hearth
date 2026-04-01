package models

import (
	"time"

	"github.com/google/uuid"
)

// CommandType represents the type of application command
type CommandType int

const (
	// CommandTypeSlash is a slash command (e.g., /help)
	CommandTypeSlash CommandType = 1
	// CommandTypeUser is a user context menu command
	CommandTypeUser CommandType = 2
	// CommandTypeMessage is a message context menu command
	CommandTypeMessage CommandType = 3
)

// OptionType represents the type of a command option
type OptionType int

const (
	OptionTypeSubcommand      OptionType = 1
	OptionTypeSubcommandGroup OptionType = 2
	OptionTypeString          OptionType = 3
	OptionTypeInteger         OptionType = 4
	OptionTypeBoolean         OptionType = 5
	OptionTypeUser            OptionType = 6
	OptionTypeChannel         OptionType = 7
	OptionTypeRole            OptionType = 8
	OptionTypeMentionable     OptionType = 9
	OptionTypeNumber          OptionType = 10
	OptionTypeAttachment      OptionType = 11
)

// CommandOption represents a parameter for a slash command
type CommandOption struct {
	Type         OptionType       `json:"type"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Required     bool             `json:"required,omitempty"`
	Choices      []*CommandChoice `json:"choices,omitempty"`
	Options      []*CommandOption `json:"options,omitempty"`
	ChannelTypes []int            `json:"channel_types,omitempty"`
	MinValue     *float64         `json:"min_value,omitempty"`
	MaxValue     *float64         `json:"max_value,omitempty"`
	MinLength    *int             `json:"min_length,omitempty"`
	MaxLength    *int             `json:"max_length,omitempty"`
	Autocomplete bool             `json:"autocomplete,omitempty"`
}

// CommandChoice represents a predefined choice for a string/integer option
type CommandChoice struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// SlashCommand represents an application command (slash command)
type SlashCommand struct {
	ID          uuid.UUID           `json:"id"`
	Type        CommandType         `json:"type"`
	AppID       uuid.UUID           `json:"application_id"`
	ServerID    *uuid.UUID          `json:"guild_id,omitempty"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Options     []*CommandOption    `json:"options,omitempty"`
	Permissions *CommandPermissions `json:"permissions,omitempty"`
	Version     string              `json:"version"`
	CreatorID   *uuid.UUID          `json:"creator_id,omitempty"`
	Default     bool                `json:"default_permission,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// CommandPermissions represents permission overrides for a command
type CommandPermissions struct {
	Overrides []*CommandPermissionOverride `json:"overrides,omitempty"`
}

// CommandPermissionOverride represents a permission override for a role or user
type CommandPermissionOverride struct {
	ID     uuid.UUID `json:"id"`
	Type   int       `json:"type"` // 1 = role, 2 = user
	Allow  bool      `json:"allow"`
	Denial bool      `json:"deny"`
}

// InteractionType represents the type of interaction
type InteractionType int

const (
	InteractionTypePing                InteractionType = 1
	InteractionTypeApplicationCommand  InteractionType = 2
	InteractionTypeMessageComponent    InteractionType = 3
	InteractionTypeApplicationCommand2 InteractionType = 2 // alias
	InteractionTypeAutocomplete        InteractionType = 4
	InteractionTypeModalSubmit         InteractionType = 5
)

// InteractionCallbackType represents how to respond to an interaction
type InteractionCallbackType int

const (
	// CallbackTypePong is an ACK for InteractionTypePing
	CallbackTypePong InteractionCallbackType = 1
	// CallbackTypeChannelMessage creates a message in the channel with the provided content
	CallbackTypeChannelMessage InteractionCallbackType = 4
	// CallbackTypeDeferredChannelMessage acknowledges the interaction and shows a loading state
	CallbackTypeDeferredChannelMessage InteractionCallbackType = 5
	// CallbackTypeDeferredUpdateMessage acknowledges a message update (buttons/selects)
	CallbackTypeDeferredUpdateMessage InteractionCallbackType = 6
	// CallbackTypeUpdateMessage edits the message the component is attached to
	CallbackTypeUpdateMessage InteractionCallbackType = 7
	// CallbackTypeAutocompleteResult returns autocomplete suggestions
	CallbackTypeAutocompleteResult InteractionCallbackType = 8
	// CallbackTypeModal creates a modal popup
	CallbackTypeModal InteractionCallbackType = 9

	// Additional callback types for interactive components
	// CallbackTypeMessageUpdate updates the message (edit)
	CallbackTypeMessageUpdate InteractionCallbackType = 10
	// CallbackTypeMessageDelete deletes the message
	CallbackTypeMessageDelete InteractionCallbackType = 11
	// CallbackTypeModalOpen opens a modal dialog (alias for CallbackTypeModal)
	CallbackTypeModalOpen InteractionCallbackType = 12
)

// InteractionCallbackData contains the response data for an interaction
type InteractionCallbackData struct {
	Content    *string          `json:"content,omitempty"`
	Embeds     []Embed          `json:"embeds,omitempty"`
	Components []interface{}    `json:"components,omitempty"`
	Flags      int              `json:"flags,omitempty"`
	Choices    []*CommandChoice `json:"choices,omitempty"`
	Title      *string          `json:"title,omitempty"`
	CustomID   *string          `json:"custom_id,omitempty"`
	Rows       []interface{}    `json:"rows,omitempty"`
}

// CommandInteractionData is the parsed data from a slash command interaction
type CommandInteractionData struct {
	ID       uuid.UUID                `json:"id"`
	Name     string                   `json:"name"`
	Type     CommandType              `json:"type"`
	Options  []*ResolvedCommandOption `json:"options,omitempty"`
	TargetID *uuid.UUID               `json:"target_id,omitempty"`
}

// ResolvedCommandOption is a command option with a resolved value
type ResolvedCommandOption struct {
	Name    string                   `json:"name"`
	Type    OptionType               `json:"type"`
	Value   interface{}              `json:"value"`
	Options []*ResolvedCommandOption `json:"options,omitempty"`
}

// Interaction represents an incoming interaction from a user
type Interaction struct {
	ID           uuid.UUID       `json:"id"`
	Type         InteractionType `json:"type"`
	Data         interface{}     `json:"data,omitempty"`
	UserID       uuid.UUID       `json:"user_id"`
	ServerID     *uuid.UUID      `json:"guild_id,omitempty"`
	ChannelID    uuid.UUID       `json:"channel_id"`
	Member       *Member         `json:"member,omitempty"`
	Token        string          `json:"token"`
	Version      int             `json:"version,omitempty"`
	AppID        uuid.UUID       `json:"application_id"`
	Entitlements []Entitlement   `json:"entitlements,omitempty"`
	Message      *Message        `json:"message,omitempty"`
}

// InteractionResponse represents a response to an interaction
type InteractionResponse struct {
	Type InteractionCallbackType  `json:"type"`
	Data *InteractionCallbackData `json:"data,omitempty"`
}

// Entitlement represents a user's entitlement to an app
type Entitlement struct {
	ID            uuid.UUID  `json:"id"`
	Type          int        `json:"type"`
	SKUID         uuid.UUID  `json:"sku_id"`
	ApplicationID uuid.UUID  `json:"application_id"`
	UserID        uuid.UUID  `json:"user_id"`
	ServerID      *uuid.UUID `json:"guild_id,omitempty"`
}

// CommandExecution represents a logged command execution
type CommandExecution struct {
	ID           uuid.UUID  `json:"id"`
	CommandID    uuid.UUID  `json:"command_id"`
	AppID        uuid.UUID  `json:"application_id"`
	ServerID     *uuid.UUID `json:"guild_id,omitempty"`
	ChannelID    uuid.UUID  `json:"channel_id"`
	UserID       uuid.UUID  `json:"user_id"`
	Options      string     `json:"options"` // JSON serialized
	ResponseTime int64      `json:"response_time_ms"`
	Status       string     `json:"status"` // success, error, denied
	ErrorMsg     *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ValidateCommand validates a slash command's structure
func (c *SlashCommand) Validate() error {
	if c.Name == "" {
		return NewValidationError("command name is required")
	}
	if len(c.Name) < 1 || len(c.Name) > 32 {
		return NewValidationError("command name must be 1-32 characters")
	}
	if c.Description == "" {
		return NewValidationError("command description is required")
	}
	if len(c.Description) > 100 {
		return NewValidationError("command description must be 100 characters or less")
	}
	// Validate options recursively
	for _, opt := range c.Options {
		if err := validateOption(opt); err != nil {
			return err
		}
	}
	return nil
}

func validateOption(opt *CommandOption) error {
	if opt.Name == "" {
		return NewValidationError("option name is required")
	}
	if len(opt.Name) < 1 || len(opt.Name) > 32 {
		return NewValidationError("option name must be 1-32 characters")
	}
	if opt.Description == "" {
		return NewValidationError("option description is required")
	}
	if len(opt.Options) > 0 {
		if opt.Type != OptionTypeSubcommand && opt.Type != OptionTypeSubcommandGroup {
			return NewValidationError("only subcommand/subcommandGroup can have nested options")
		}
		for _, sub := range opt.Options {
			if err := validateOption(sub); err != nil {
				return err
			}
		}
	}
	return nil
}
