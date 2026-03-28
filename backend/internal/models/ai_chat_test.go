package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConversationMessageRoleValid(t *testing.T) {
	validRoles := []ConversationMessageRole{
		ConvRoleSystem, ConvRoleUser, ConvRoleAssistant, ConvRoleTool,
	}
	for _, r := range validRoles {
		if !r.Valid() {
			t.Errorf("expected %s to be valid", r)
		}
	}

	invalid := ConversationMessageRole("invalid")
	if invalid.Valid() {
		t.Error("expected invalid role to be invalid")
	}
}

func TestConversationMessageToAPIMessage(t *testing.T) {
	name := "test-tool"
	msg := &ConversationMessage{
		ID:             uuid.New(),
		ConversationID: uuid.New(),
		Role:           ConvRoleAssistant,
		Content:        "hello world",
		Name:           &name,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	api := msg.ToAPIMessage()
	if api.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %s", api.Role)
	}
	if api.Content != "hello world" {
		t.Errorf("expected content 'hello world', got %s", api.Content)
	}
	if api.Name != "test-tool" {
		t.Errorf("expected name 'test-tool', got %s", api.Name)
	}
}

func TestConversationMessageToAPIMessageNilName(t *testing.T) {
	msg := &ConversationMessage{
		Role:    ConvRoleUser,
		Content: "test",
		Name:    nil,
	}

	api := msg.ToAPIMessage()
	if api.Name != "" {
		t.Errorf("expected empty name for nil pointer, got %s", api.Name)
	}
}

func TestToolCallsJSONValueAndScan(t *testing.T) {
	// Test nil value
	var nilCalls ToolCallsJSON
	val, err := nilCalls.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value for nil ToolCallsJSON")
	}

	// Test with data
	calls := ToolCallsJSON{
		{ID: "1", Type: "function", Function: ConversationToolCallFunction{Name: "test", Arguments: "{}"}},
	}
	val, err = calls.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil value")
	}

	// Test Scan with valid data
	var scanned ToolCallsJSON
	err = scanned.Scan(val.([]byte))
	if err != nil {
		t.Fatalf("unexpected error scanning: %v", err)
	}
	if len(scanned) != 1 || scanned[0].ID != "1" {
		t.Errorf("unexpected scanned value: %v", scanned)
	}

	// Test Scan with nil
	var nilScanned ToolCallsJSON
	err = nilScanned.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nilScanned != nil {
		t.Error("expected nil after scanning nil")
	}

	// Test Scan with non-byte type
	var nonByte ToolCallsJSON
	err = nonByte.Scan("not bytes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitialMessagesJSONValueAndScan(t *testing.T) {
	// Test nil value
	var nilMsgs InitialMessagesJSON
	val, err := nilMsgs.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value for nil InitialMessagesJSON")
	}

	// Test with data
	msgs := InitialMessagesJSON{
		{Role: "system", Content: "You are helpful"},
	}
	val, err = msgs.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil value")
	}

	// Test Scan
	var scanned InitialMessagesJSON
	err = scanned.Scan(val.([]byte))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scanned) != 1 || scanned[0].Role != "system" {
		t.Errorf("unexpected scanned value: %v", scanned)
	}

	// Test Scan nil
	var nilScanned InitialMessagesJSON
	err = nilScanned.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nilScanned != nil {
		t.Error("expected nil after scanning nil")
	}

	// Test Scan non-byte type
	var nonByte InitialMessagesJSON
	err = nonByte.Scan(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSuggestedPromptsJSONValueAndScan(t *testing.T) {
	// Test nil value
	var nilPrompts SuggestedPromptsJSON
	val, err := nilPrompts.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value for nil SuggestedPromptsJSON")
	}

	// Test with data
	prompts := SuggestedPromptsJSON{"hello", "world"}
	val, err = prompts.Value()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil value")
	}

	// Test Scan
	var scanned SuggestedPromptsJSON
	err = scanned.Scan(val.([]byte))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scanned) != 2 || scanned[0] != "hello" {
		t.Errorf("unexpected scanned value: %v", scanned)
	}

	// Test Scan nil
	var nilScanned SuggestedPromptsJSON
	err = nilScanned.Scan(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nilScanned != nil {
		t.Error("expected nil after scanning nil")
	}

	// Test Scan non-byte type
	var nonByte SuggestedPromptsJSON
	err = nonByte.Scan(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
