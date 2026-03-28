package models

import (
	"testing"

	"github.com/google/uuid"
)

func TestSlashCommandValidate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     SlashCommand
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid command",
			cmd:     SlashCommand{ID: uuid.New(), Name: "test", Description: "A test command"},
			wantErr: false,
		},
		{
			name:    "empty name",
			cmd:     SlashCommand{ID: uuid.New(), Name: "", Description: "desc"},
			wantErr: true,
			errMsg:  "command name is required",
		},
		{
			name:    "name too long",
			cmd:     SlashCommand{ID: uuid.New(), Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Description: "desc"},
			wantErr: true,
			errMsg:  "command name must be 1-32 characters",
		},
		{
			name:    "empty description",
			cmd:     SlashCommand{ID: uuid.New(), Name: "test", Description: ""},
			wantErr: true,
			errMsg:  "command description is required",
		},
		{
			name: "description too long",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			wantErr: true,
			errMsg:  "command description must be 100 characters or less",
		},
		{
			name: "valid with options",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "A test command",
				Options: []*CommandOption{
					{Type: OptionTypeString, Name: "input", Description: "User input"},
				},
			},
			wantErr: false,
		},
		{
			name: "option with empty name",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "A test command",
				Options: []*CommandOption{
					{Type: OptionTypeString, Name: "", Description: "desc"},
				},
			},
			wantErr: true,
			errMsg:  "option name is required",
		},
		{
			name: "option with empty description",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "A test command",
				Options: []*CommandOption{
					{Type: OptionTypeString, Name: "input", Description: ""},
				},
			},
			wantErr: true,
			errMsg:  "option description is required",
		},
		{
			name: "nested options valid subcommand",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "A test command",
				Options: []*CommandOption{
					{
						Type:        OptionTypeSubcommand,
						Name:        "sub",
						Description: "A subcommand",
						Options: []*CommandOption{
							{Type: OptionTypeString, Name: "arg", Description: "An argument"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "nested options invalid non-subcommand",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "A test command",
				Options: []*CommandOption{
					{
						Type:        OptionTypeString,
						Name:        "str",
						Description: "A string",
						Options: []*CommandOption{
							{Type: OptionTypeString, Name: "nested", Description: "Nested"},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "only subcommand/subcommandGroup can have nested options",
		},
		{
			name: "subcommand group with nested subcommands",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "A test command",
				Options: []*CommandOption{
					{
						Type:        OptionTypeSubcommandGroup,
						Name:        "group",
						Description: "A group",
						Options: []*CommandOption{
							{Type: OptionTypeSubcommand, Name: "sub", Description: "A sub"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "option name too long",
			cmd: SlashCommand{
				ID:          uuid.New(),
				Name:        "test",
				Description: "A test command",
				Options: []*CommandOption{
					{Type: OptionTypeString, Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Description: "desc"},
				},
			},
			wantErr: true,
			errMsg:  "option name must be 1-32 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
