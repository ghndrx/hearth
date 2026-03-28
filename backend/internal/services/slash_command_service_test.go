package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"hearth/internal/models"
)

// MockSlashCommandRepository implements SlashCommandRepository for testing
type MockSlashCommandRepository struct {
	commands     map[uuid.UUID]*models.SlashCommand
	execLogs     []*models.CommandExecution
	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
}

func NewMockSlashCommandRepository() *MockSlashCommandRepository {
	return &MockSlashCommandRepository{
		commands: make(map[uuid.UUID]*models.SlashCommand),
		execLogs: make([]*models.CommandExecution, 0),
	}
}

func (m *MockSlashCommandRepository) Create(ctx context.Context, cmd *models.SlashCommand) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.commands[cmd.ID] = cmd
	return nil
}

func (m *MockSlashCommandRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SlashCommand, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cmd, ok := m.commands[id]
	if !ok {
		return nil, nil
	}
	return cmd, nil
}

func (m *MockSlashCommandRepository) GetByAppID(ctx context.Context, appID uuid.UUID) ([]*models.SlashCommand, error) {
	var result []*models.SlashCommand
	for _, cmd := range m.commands {
		if cmd.AppID == appID {
			result = append(result, cmd)
		}
	}
	return result, nil
}

func (m *MockSlashCommandRepository) GetByServerID(ctx context.Context, serverID uuid.UUID) ([]*models.SlashCommand, error) {
	var result []*models.SlashCommand
	for _, cmd := range m.commands {
		// Include global commands (server_id IS NULL) and server-specific commands
		if cmd.ServerID == nil || (cmd.ServerID != nil && *cmd.ServerID == serverID) {
			result = append(result, cmd)
		}
	}
	return result, nil
}

func (m *MockSlashCommandRepository) GetByName(ctx context.Context, appID uuid.UUID, name string, serverID *uuid.UUID) (*models.SlashCommand, error) {
	for _, cmd := range m.commands {
		if cmd.AppID == appID && cmd.Name == name {
			if serverID == nil && cmd.ServerID == nil {
				return cmd, nil
			}
			if serverID != nil && cmd.ServerID != nil && *serverID == *cmd.ServerID {
				return cmd, nil
			}
		}
	}
	return nil, nil
}

func (m *MockSlashCommandRepository) Update(ctx context.Context, cmd *models.SlashCommand) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.commands[cmd.ID] = cmd
	return nil
}

func (m *MockSlashCommandRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.commands, id)
	return nil
}

func (m *MockSlashCommandRepository) CreateExecutionLog(ctx context.Context, log *models.CommandExecution) error {
	m.execLogs = append(m.execLogs, log)
	return nil
}

func TestSlashCommand_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *models.SlashCommand
		wantErr bool
	}{
		{
			name: "valid command",
			cmd: &models.SlashCommand{
				Name:        "test",
				Description: "A test command",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			cmd: &models.SlashCommand{
				Name:        "",
				Description: "A test command",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			cmd: &models.SlashCommand{
				Name:        "this-is-a-very-long-command-name-that-exceeds-32-chars",
				Description: "A test command",
			},
			wantErr: true,
		},
		{
			name: "empty description",
			cmd: &models.SlashCommand{
				Name:        "test",
				Description: "",
			},
			wantErr: true,
		},
		{
			name: "description too long",
			cmd: &models.SlashCommand{
				Name:        "test",
				Description: "This is a very long description that exceeds the maximum allowed length of one hundred characters for a command description",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("SlashCommand.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlashCommandService_RegisterCommand(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	t.Run("registers valid command", func(t *testing.T) {
		cmd := &models.SlashCommand{
			Name:        "ping",
			Description: "Check bot latency",
			Options: []*models.CommandOption{
				{
					Name:        "value",
					Description: "Optional value",
					Type:        models.OptionTypeString,
					Required:    false,
				},
			},
		}

		err := service.RegisterCommand(ctx, appID, cmd)
		if err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
		if cmd.ID == uuid.Nil {
			t.Error("Command ID was not set")
		}
		if cmd.AppID != appID {
			t.Errorf("Command AppID = %v, want %v", cmd.AppID, appID)
		}
		if cmd.Version == "" {
			t.Error("Command version was not set")
		}
	})

	t.Run("rejects duplicate name", func(t *testing.T) {
		cmd := &models.SlashCommand{
			Name:        "ping",
			Description: "Duplicate command",
		}
		err := service.RegisterCommand(ctx, appID, cmd)
		if err == nil {
			t.Fatal("Expected error for duplicate command name")
		}
	})

	t.Run("rejects invalid command", func(t *testing.T) {
		cmd := &models.SlashCommand{
			Name:        "",
			Description: "Invalid command",
		}
		err := service.RegisterCommand(ctx, appID, cmd)
		if err == nil {
			t.Fatal("Expected error for invalid command")
		}
	})
}

func TestSlashCommandService_GetCommands(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	// Register some commands
	for _, name := range []string{"alpha", "beta", "gamma"} {
		cmd := &models.SlashCommand{
			Name:        name,
			Description: "Test command " + name,
		}
		if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
	}

	t.Run("retrieves all commands for app", func(t *testing.T) {
		commands, err := service.GetCommands(ctx, appID)
		if err != nil {
			t.Fatalf("GetCommands() error = %v", err)
		}
		if len(commands) != 3 {
			t.Errorf("GetCommands() returned %d commands, want 3", len(commands))
		}
	})

	t.Run("retrieves specific command", func(t *testing.T) {
		// Get the first command ID
		commands, _ := service.GetCommands(ctx, appID)
		if len(commands) == 0 {
			t.Fatal("No commands found")
		}
		cmdID := commands[0].ID

		cmd, err := service.GetCommand(ctx, appID, cmdID)
		if err != nil {
			t.Fatalf("GetCommand() error = %v", err)
		}
		if cmd.Name != commands[0].Name {
			t.Errorf("GetCommand() name = %v, want %v", cmd.Name, commands[0].Name)
		}
	})
}

func TestSlashCommandService_UpdateCommand(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	// Register a command
	cmd := &models.SlashCommand{
		Name:        "oldname",
		Description: "Old description",
	}
	if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}

	t.Run("updates command", func(t *testing.T) {
		updated := &models.SlashCommand{
			Name:        "newname",
			Description: "New description",
		}
		err := service.UpdateCommand(ctx, appID, cmd.ID, updated)
		if err != nil {
			t.Fatalf("UpdateCommand() error = %v", err)
		}

		cmd, _ := service.GetCommand(ctx, appID, cmd.ID)
		if cmd.Name != "newname" {
			t.Errorf("UpdateCommand() name = %v, want newname", cmd.Name)
		}
		if cmd.Description != "New description" {
			t.Errorf("UpdateCommand() description = %v, want New description", cmd.Description)
		}
		// Version should change
		if cmd.Version == "" {
			t.Error("Version was not updated")
		}
	})

	t.Run("rejects update for wrong app", func(t *testing.T) {
		wrongAppID := uuid.New()
		updated := &models.SlashCommand{
			Name:        "name",
			Description: "Desc",
		}
		err := service.UpdateCommand(ctx, wrongAppID, cmd.ID, updated)
		if err == nil {
			t.Fatal("Expected error for wrong app")
		}
	})
}

func TestSlashCommandService_DeleteCommand(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	// Register a command
	cmd := &models.SlashCommand{
		Name:        "deleteable",
		Description: "Will be deleted",
	}
	if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}

	t.Run("deletes command", func(t *testing.T) {
		err := service.DeleteCommand(ctx, appID, cmd.ID)
		if err != nil {
			t.Fatalf("DeleteCommand() error = %v", err)
		}

		// Verify it's gone
		_, getErr := service.GetCommand(ctx, appID, cmd.ID)
		if getErr == nil {
			t.Error("Command still exists after deletion")
		}
	})
}

func TestSlashCommand_ValidateOption(t *testing.T) {
	tests := []struct {
		name    string
		opt     *models.CommandOption
		wantErr bool
	}{
		{
			name: "valid option",
			opt: &models.CommandOption{
				Name:        "username",
				Description: "The user to look up",
				Type:        models.OptionTypeString,
				Required:    true,
			},
			wantErr: false,
		},
		{
			name: "valid subcommand with nested options",
			opt: &models.CommandOption{
				Name:        "moderation",
				Description: "Moderation actions",
				Type:        models.OptionTypeSubcommand,
				Options: []*models.CommandOption{
					{
						Name:        "action",
						Description: "What to do",
						Type:        models.OptionTypeString,
						Required:    true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "nested option in non-subcommand",
			opt: &models.CommandOption{
				Name:        "bad",
				Description: "Invalid",
				Type:        models.OptionTypeString,
				Options: []*models.CommandOption{
					{
						Name:        "nested",
						Description: "Should fail",
						Type:        models.OptionTypeString,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &models.SlashCommand{
				Name:        "test",
				Description: "Test",
				Options:     []*models.CommandOption{tt.opt},
			}
			err := cmd.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandOption_Types(t *testing.T) {
	// Verify option type constants
	if int(models.OptionTypeSubcommand) != 1 {
		t.Errorf("OptionTypeSubcommand = %d, want 1", models.OptionTypeSubcommand)
	}
	if int(models.OptionTypeSubcommandGroup) != 2 {
		t.Errorf("OptionTypeSubcommandGroup = %d, want 2", models.OptionTypeSubcommandGroup)
	}
	if int(models.OptionTypeString) != 3 {
		t.Errorf("OptionTypeString = %d, want 3", models.OptionTypeString)
	}
	if int(models.OptionTypeInteger) != 4 {
		t.Errorf("OptionTypeInteger = %d, want 4", models.OptionTypeInteger)
	}
	if int(models.OptionTypeBoolean) != 5 {
		t.Errorf("OptionTypeBoolean = %d, want 5", models.OptionTypeBoolean)
	}
	if int(models.OptionTypeUser) != 6 {
		t.Errorf("OptionTypeUser = %d, want 6", models.OptionTypeUser)
	}
	if int(models.OptionTypeChannel) != 7 {
		t.Errorf("OptionTypeChannel = %d, want 7", models.OptionTypeChannel)
	}
	if int(models.OptionTypeRole) != 8 {
		t.Errorf("OptionTypeRole = %d, want 8", models.OptionTypeRole)
	}
	if int(models.OptionTypeMentionable) != 9 {
		t.Errorf("OptionTypeMentionable = %d, want 9", models.OptionTypeMentionable)
	}
	if int(models.OptionTypeNumber) != 10 {
		t.Errorf("OptionTypeNumber = %d, want 10", models.OptionTypeNumber)
	}
	if int(models.OptionTypeAttachment) != 11 {
		t.Errorf("OptionTypeAttachment = %d, want 11", models.OptionTypeAttachment)
	}
}

func TestInteractionResponse_Types(t *testing.T) {
	if int(models.CallbackTypePong) != 1 {
		t.Errorf("CallbackTypePong = %d, want 1", models.CallbackTypePong)
	}
	if int(models.CallbackTypeChannelMessage) != 4 {
		t.Errorf("CallbackTypeChannelMessage = %d, want 4", models.CallbackTypeChannelMessage)
	}
	if int(models.CallbackTypeDeferredChannelMessage) != 5 {
		t.Errorf("CallbackTypeDeferredChannelMessage = %d, want 5", models.CallbackTypeDeferredChannelMessage)
	}
	if int(models.CallbackTypeDeferredUpdateMessage) != 6 {
		t.Errorf("CallbackTypeDeferredUpdateMessage = %d, want 6", models.CallbackTypeDeferredUpdateMessage)
	}
	if int(models.CallbackTypeUpdateMessage) != 7 {
		t.Errorf("CallbackTypeUpdateMessage = %d, want 7", models.CallbackTypeUpdateMessage)
	}
	if int(models.CallbackTypeAutocompleteResult) != 8 {
		t.Errorf("CallbackTypeAutocompleteResult = %d, want 8", models.CallbackTypeAutocompleteResult)
	}
	if int(models.CallbackTypeModal) != 9 {
		t.Errorf("CallbackTypeModal = %d, want 9", models.CallbackTypeModal)
	}
}

func TestSlashCommand_VersionIsGenerated(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	cmd := &models.SlashCommand{
		Name:        "versiontest",
		Description: "Test version generation",
	}

	if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}

	originalVersion := cmd.Version

	// Update the command
	updated := &models.SlashCommand{
		Name:        "versiontest",
		Description: "Updated description",
	}
	if err := service.UpdateCommand(ctx, appID, cmd.ID, updated); err != nil {
		t.Fatalf("UpdateCommand() error = %v", err)
	}

	cmd, _ = service.GetCommand(ctx, appID, cmd.ID)
	if cmd.Version == originalVersion {
		t.Error("Version was not updated after command modification")
	}
}

func TestSlashCommandService_BulkRegister(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	commands := []*models.SlashCommand{
		{Name: "cmd1", Description: "Command 1"},
		{Name: "cmd2", Description: "Command 2"},
		{Name: "cmd3", Description: "Command 3"},
	}

	for _, cmd := range commands {
		if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
	}

	result, err := service.GetCommands(ctx, appID)
	if err != nil {
		t.Fatalf("GetCommands() error = %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 commands, got %d", len(result))
	}
}

func TestSlashCommand_DefaultPermission(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	t.Run("default_permission defaults to true", func(t *testing.T) {
		cmd := &models.SlashCommand{
			Name:        "defaultperm",
			Description: "Test default permission",
		}
		if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
		if !cmd.Default {
			t.Error("Default permission should be true by default")
		}
	})
}

func TestSlashCommand_OptionsSerialization(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()

	options := []*models.CommandOption{
		{
			Name:        "action",
			Description: "The action to perform",
			Type:        models.OptionTypeString,
			Required:    true,
			Choices: []*models.CommandChoice{
				{Name: "Kick", Value: "kick"},
				{Name: "Ban", Value: "ban"},
				{Name: "Warn", Value: "warn"},
			},
		},
		{
			Name:         "user",
			Description:  "The target user",
			Type:         models.OptionTypeUser,
			Required:     true,
			Autocomplete: false,
		},
		{
			Name:        "reason",
			Description: "Optional reason",
			Type:        models.OptionTypeString,
			Required:    false,
		},
	}

	cmd := &models.SlashCommand{
		Name:        "mod",
		Description: "Moderation command",
		Options:     options,
	}

	if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
		t.Fatalf("RegisterCommand() error = %v", err)
	}

	// Retrieve and verify
	retrieved, err := service.GetCommand(ctx, appID, cmd.ID)
	if err != nil {
		t.Fatalf("GetCommand() error = %v", err)
	}

	if len(retrieved.Options) != 3 {
		t.Errorf("Expected 3 options, got %d", len(retrieved.Options))
	}

	actionOpt := retrieved.Options[0]
	if actionOpt.Name != "action" {
		t.Errorf("Action option name = %v, want action", actionOpt.Name)
	}
	if len(actionOpt.Choices) != 3 {
		t.Errorf("Expected 3 choices, got %d", len(actionOpt.Choices))
	}
}

func TestSlashCommand_ServerScopedCommand(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()
	serverID := uuid.New()

	t.Run("register server-specific command", func(t *testing.T) {
		cmd := &models.SlashCommand{
			Name:        "servercmd",
			Description: "Server-specific command",
			ServerID:    &serverID,
		}
		if err := service.RegisterCommand(ctx, appID, cmd); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
		if cmd.ServerID == nil || *cmd.ServerID != serverID {
			t.Error("ServerID was not set correctly")
		}
	})

	t.Run("global and server command can have same name", func(t *testing.T) {
		// Register global command
		globalCmd := &models.SlashCommand{
			Name:        "hello",
			Description: "Global hello",
		}
		if err := service.RegisterCommand(ctx, appID, globalCmd); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}

		// Register server-specific command with same name
		serverCmd := &models.SlashCommand{
			Name:        "hello",
			Description: "Server hello",
			ServerID:    &serverID,
		}
		if err := service.RegisterCommand(ctx, appID, serverCmd); err != nil {
			t.Fatalf("RegisterCommand() error = %v", err)
		}
	})
}

func TestCommandExecution_Logging(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()

	cmd := &models.SlashCommand{
		Name:        "logtest",
		Description: "Test execution logging",
	}
	service.RegisterCommand(ctx, uuid.New(), cmd)

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypeApplicationCommand,
		UserID:    uuid.New(),
		ChannelID: uuid.New(),
		AppID:    uuid.New(),
		Token:    "test-token",
		Data: &models.CommandInteractionData{
			ID:   cmd.ID,
			Name: "logtest",
		},
	}

	_, _ = service.ExecuteCommand(ctx, interaction)

	if len(repo.execLogs) != 1 {
		t.Errorf("Expected 1 execution log, got %d", len(repo.execLogs))
	}

	log := repo.execLogs[0]
	if log.CommandID != cmd.ID {
		t.Errorf("Log CommandID = %v, want %v", log.CommandID, cmd.ID)
	}
	if log.Status == "" {
		t.Error("Log status was not set")
	}
	if log.ResponseTime < 0 {
		t.Error("Response time should be non-negative")
	}
}

func TestSlashCommandService_ExecuteCommand_NoCommand(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()

	interaction := &models.Interaction{
		ID:        uuid.New(),
		Type:      models.InteractionTypeApplicationCommand,
		UserID:    uuid.New(),
		ChannelID: uuid.New(),
		AppID:    uuid.New(),
		Token:    "test-token",
		Data: &models.CommandInteractionData{
			ID:   uuid.New(),
			Name: "nonexistent",
		},
	}

	resp, err := service.ExecuteCommand(ctx, interaction)
	if err != nil {
		t.Fatalf("ExecuteCommand() error = %v", err)
	}
	// When command not found, it logs the execution and returns nil response
	// This is because logExecution returns nil
	_ = resp // response may be nil when command not found - logged internally
}

func TestInteractionCallbackData_Ephemeral(t *testing.T) {
	// Test that ephemeral flag (1 << 6 = 64) can be set
	data := &models.InteractionCallbackData{
		Content: strPtr("Secret message"),
		Flags:   1 << 6, // Ephemeral flag
	}

	if data.Flags != 64 {
		t.Errorf("Ephemeral flag = %d, want 64", data.Flags)
	}
}

func TestSlashCommand_CommandTypes(t *testing.T) {
	if int(models.CommandTypeSlash) != 1 {
		t.Errorf("CommandTypeSlash = %d, want 1", models.CommandTypeSlash)
	}
	if int(models.CommandTypeUser) != 2 {
		t.Errorf("CommandTypeUser = %d, want 2", models.CommandTypeUser)
	}
	if int(models.CommandTypeMessage) != 3 {
		t.Errorf("CommandTypeMessage = %d, want 3", models.CommandTypeMessage)
	}
}

func TestInteractionTypes(t *testing.T) {
	if int(models.InteractionTypePing) == 0 {
		t.Error("InteractionTypePing should not be zero")
	}
	if int(models.InteractionTypeApplicationCommand) == 0 {
		t.Error("InteractionTypeApplicationCommand should not be zero")
	}
	if int(models.InteractionTypeMessageComponent) == 0 {
		t.Error("InteractionTypeMessageComponent should not be zero")
	}
	if int(models.InteractionTypeAutocomplete) == 0 {
		t.Error("InteractionTypeAutocomplete should not be zero")
	}
	if int(models.InteractionTypeModalSubmit) == 0 {
		t.Error("InteractionTypeModalSubmit should not be zero")
	}
}

func TestSlashCommandService_GetServerCommands(t *testing.T) {
	repo := NewMockSlashCommandRepository()
	service := NewSlashCommandService(repo, nil, nil, nil)
	ctx := context.Background()
	appID := uuid.New()
	serverID := uuid.New()

	// Register global command
	globalCmd := &models.SlashCommand{
		Name:        "global",
		Description: "Global command",
	}
	service.RegisterCommand(ctx, appID, globalCmd)

	// Register server-specific command
	serverCmd := &models.SlashCommand{
		Name:        "serveronly",
		Description: "Server command",
		ServerID:    &serverID,
	}
	service.RegisterCommand(ctx, appID, serverCmd)

	// Get server commands (should include both)
	commands, err := service.GetServerCommands(ctx, serverID)
	if err != nil {
		t.Fatalf("GetServerCommands() error = %v", err)
	}
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}
}
