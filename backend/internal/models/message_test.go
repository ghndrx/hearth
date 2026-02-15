package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMessageTypes(t *testing.T) {
	expectedTypes := map[MessageType]string{
		MessageTypeDefault:           "default",
		MessageTypeReply:             "reply",
		MessageTypeRecipientAdd:      "recipient_add",
		MessageTypeRecipientRemove:   "recipient_remove",
		MessageTypeCall:              "call",
		MessageTypeChannelNameChange: "channel_name_change",
		MessageTypeChannelIconChange: "channel_icon_change",
		MessageTypePinned:            "pinned",
		MessageTypeMemberJoin:        "member_join",
		MessageTypeThreadCreated:     "thread_created",
	}

	for msgType, expected := range expectedTypes {
		if string(msgType) != expected {
			t.Errorf("expected message type %s, got %s", expected, string(msgType))
		}
	}
}

func TestMessageFlags(t *testing.T) {
	// Verify flag bit positions
	expectedFlags := map[int]int{
		MessageFlagCrossposted:      1 << 0,
		MessageFlagIsCrosspost:      1 << 1,
		MessageFlagSuppressEmbeds:   1 << 2,
		MessageFlagSourceMsgDeleted: 1 << 3,
		MessageFlagUrgent:           1 << 4,
		MessageFlagHasThread:        1 << 5,
		MessageFlagEphemeral:        1 << 6,
		MessageFlagLoading:          1 << 7,
		MessageFlagFailedToMention:  1 << 8,
	}

	for flag, expected := range expectedFlags {
		if flag != expected {
			t.Errorf("expected flag %d, got %d", expected, flag)
		}
	}

	// Test combining flags
	combined := MessageFlagCrossposted | MessageFlagUrgent | MessageFlagHasThread
	if combined&MessageFlagCrossposted == 0 {
		t.Error("combined should include Crossposted")
	}
	if combined&MessageFlagUrgent == 0 {
		t.Error("combined should include Urgent")
	}
	if combined&MessageFlagHasThread == 0 {
		t.Error("combined should include HasThread")
	}
	if combined&MessageFlagEphemeral != 0 {
		t.Error("combined should not include Ephemeral")
	}
}

func TestMessageStruct(t *testing.T) {
	serverID := uuid.New()
	replyToID := uuid.New()
	threadID := uuid.New()
	editedAt := time.Now()

	msg := Message{
		ID:               uuid.New(),
		ChannelID:        uuid.New(),
		ServerID:         &serverID,
		AuthorID:         uuid.New(),
		Content:          "Hello, world!",
		EncryptedContent: "encrypted-data",
		Type:             MessageTypeDefault,
		ReplyToID:        &replyToID,
		ThreadID:         &threadID,
		Pinned:           true,
		TTS:              false,
		MentionEveryone:  true,
		Flags:            MessageFlagCrossposted,
		CreatedAt:        time.Now(),
		EditedAt:         &editedAt,
		Attachments:      []Attachment{},
		Embeds:           []Embed{},
		Reactions:        []Reaction{},
		Mentions:         []uuid.UUID{uuid.New()},
		MentionRoles:     []uuid.UUID{uuid.New()},
	}

	if msg.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got %s", msg.Content)
	}
	if msg.Type != MessageTypeDefault {
		t.Errorf("expected type 'default', got %s", msg.Type)
	}
	if !msg.Pinned {
		t.Error("expected message to be pinned")
	}
	if !msg.MentionEveryone {
		t.Error("expected mention everyone")
	}
	if msg.Flags != MessageFlagCrossposted {
		t.Errorf("expected flags %d, got %d", MessageFlagCrossposted, msg.Flags)
	}
}

func TestAttachment(t *testing.T) {
	proxyURL := "https://proxy.example.com/file.png"
	contentType := "image/png"
	width := 1920
	height := 1080

	attachment := Attachment{
		ID:           uuid.New(),
		MessageID:    uuid.New(),
		Filename:     "image.png",
		URL:          "https://example.com/file.png",
		ProxyURL:     &proxyURL,
		Size:         1024000,
		ContentType:  &contentType,
		Width:        &width,
		Height:       &height,
		Ephemeral:    true,
		Encrypted:    true,
		EncryptedKey: "key-data",
		IV:           "iv-data",
		CreatedAt:    time.Now(),
	}

	if attachment.Filename != "image.png" {
		t.Errorf("expected filename 'image.png', got %s", attachment.Filename)
	}
	if attachment.Size != 1024000 {
		t.Errorf("expected size 1024000, got %d", attachment.Size)
	}
	if *attachment.Width != 1920 {
		t.Errorf("expected width 1920, got %d", *attachment.Width)
	}
	if !attachment.Encrypted {
		t.Error("expected attachment to be encrypted")
	}
}

func TestEmbed(t *testing.T) {
	title := "Test Embed"
	description := "This is a test embed"
	url := "https://example.com"
	timestamp := time.Now()
	color := 0xFF0000

	embed := Embed{
		Type:        "rich",
		Title:       &title,
		Description: &description,
		URL:         &url,
		Timestamp:   &timestamp,
		Color:       &color,
		Footer: &EmbedFooter{
			Text:    "Footer text",
			IconURL: nil,
		},
		Image: &EmbedMedia{
			URL:    "https://example.com/image.png",
			Width:  intPtr(800),
			Height: intPtr(600),
		},
		Thumbnail: &EmbedMedia{
			URL: "https://example.com/thumb.png",
		},
		Author: &EmbedAuthor{
			Name:    "Author Name",
			URL:     strPtr("https://author.example.com"),
			IconURL: nil,
		},
		Fields: []EmbedField{
			{Name: "Field 1", Value: "Value 1", Inline: true},
			{Name: "Field 2", Value: "Value 2", Inline: false},
		},
	}

	if embed.Type != "rich" {
		t.Errorf("expected type 'rich', got %s", embed.Type)
	}
	if *embed.Title != "Test Embed" {
		t.Errorf("expected title 'Test Embed', got %s", *embed.Title)
	}
	if len(embed.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(embed.Fields))
	}
	if embed.Fields[0].Inline != true {
		t.Error("expected first field to be inline")
	}
}

func TestReaction(t *testing.T) {
	reaction := Reaction{
		MessageID: uuid.New(),
		Emoji:     "👍",
		Count:     42,
		Me:        true,
	}

	if reaction.Emoji != "👍" {
		t.Errorf("expected emoji '👍', got %s", reaction.Emoji)
	}
	if reaction.Count != 42 {
		t.Errorf("expected count 42, got %d", reaction.Count)
	}
	if !reaction.Me {
		t.Error("expected me to be true")
	}
}

func TestReactionUser(t *testing.T) {
	reactionUser := ReactionUser{
		MessageID: uuid.New(),
		UserID:    uuid.New(),
		Emoji:     "❤️",
		CreatedAt: time.Now(),
	}

	if reactionUser.Emoji != "❤️" {
		t.Errorf("expected emoji '❤️', got %s", reactionUser.Emoji)
	}
}

func TestMentionType(t *testing.T) {
	if MentionTypeUser != 0 {
		t.Errorf("expected MentionTypeUser to be 0, got %d", MentionTypeUser)
	}
	if MentionTypeRole != 1 {
		t.Errorf("expected MentionTypeRole to be 1, got %d", MentionTypeRole)
	}
	if MentionTypeChannel != 2 {
		t.Errorf("expected MentionTypeChannel to be 2, got %d", MentionTypeChannel)
	}
	if MentionTypeEveryone != 3 {
		t.Errorf("expected MentionTypeEveryone to be 3, got %d", MentionTypeEveryone)
	}
	if MentionTypeHere != 4 {
		t.Errorf("expected MentionTypeHere to be 4, got %d", MentionTypeHere)
	}
}

func TestMention(t *testing.T) {
	mention := Mention{
		Type: MentionTypeUser,
		ID:   uuid.New(),
	}

	if mention.Type != MentionTypeUser {
		t.Errorf("expected type MentionTypeUser, got %d", mention.Type)
	}
}

func TestCreateMessageRequest(t *testing.T) {
	replyToID := "message-123"
	tts := true

	req := CreateMessageRequest{
		Content:   "Hello!",
		ReplyToID: &replyToID,
		TTS:       &tts,
	}

	if req.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %s", req.Content)
	}
	if *req.ReplyToID != "message-123" {
		t.Errorf("expected reply to 'message-123', got %s", *req.ReplyToID)
	}
}

func TestUpdateMessageRequest(t *testing.T) {
	content := "Updated content"

	req := UpdateMessageRequest{
		Content: &content,
	}

	if *req.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got %s", *req.Content)
	}
}

func TestMessageQueryOptions(t *testing.T) {
	before := uuid.New()
	after := uuid.New()
	around := uuid.New()

	opts := MessageQueryOptions{
		Before: &before,
		After:  &after,
		Around: &around,
		Limit:  50,
	}

	if *opts.Before != before {
		t.Errorf("expected before %v, got %v", before, *opts.Before)
	}
	if opts.Limit != 50 {
		t.Errorf("expected limit 50, got %d", opts.Limit)
	}
}

func TestPin(t *testing.T) {
	pin := Pin{
		ChannelID: uuid.New(),
		MessageID: uuid.New(),
		PinnedBy:  uuid.New(),
		PinnedAt:  time.Now(),
	}

	if pin.ChannelID == uuid.Nil {
		t.Error("expected non-nil channel ID")
	}
	if pin.MessageID == uuid.Nil {
		t.Error("expected non-nil message ID")
	}
}

func TestTypingIndicator(t *testing.T) {
	typing := TypingIndicator{
		ChannelID: uuid.New(),
		UserID:    uuid.New(),
		Timestamp: time.Now(),
	}

	if typing.ChannelID == uuid.Nil {
		t.Error("expected non-nil channel ID")
	}
	if typing.UserID == uuid.Nil {
		t.Error("expected non-nil user ID")
	}
}

func TestReadState(t *testing.T) {
	lastMsgID := uuid.New()

	state := ReadState{
		UserID:        uuid.New(),
		ChannelID:     uuid.New(),
		LastMessageID: &lastMsgID,
		MentionCount:  5,
		LastReadAt:    time.Now(),
	}

	if state.MentionCount != 5 {
		t.Errorf("expected mention count 5, got %d", state.MentionCount)
	}
	if *state.LastMessageID != lastMsgID {
		t.Errorf("expected last message ID %v, got %v", lastMsgID, *state.LastMessageID)
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}
