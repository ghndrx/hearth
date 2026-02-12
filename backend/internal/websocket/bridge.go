package websocket

import (
	"encoding/json"

	"github.com/google/uuid"

	"hearth/internal/events"
	"hearth/internal/models"
)

// EventBridge connects the domain event bus to the WebSocket hub
type EventBridge struct {
	hub *Hub
	bus *events.Bus
}

// NewEventBridge creates a new event bridge
func NewEventBridge(hub *Hub, bus *events.Bus) *EventBridge {
	bridge := &EventBridge{
		hub: hub,
		bus: bus,
	}
	bridge.registerHandlers()
	return bridge
}

// registerHandlers sets up event handlers for all domain events
func (b *EventBridge) registerHandlers() {
	// Message events
	b.bus.Subscribe(events.MessageCreated, b.onMessageCreated)
	b.bus.Subscribe(events.MessageUpdated, b.onMessageUpdated)
	b.bus.Subscribe(events.MessageDeleted, b.onMessageDeleted)
	b.bus.Subscribe(events.MessagePinned, b.onMessagePinned)

	// Reaction events
	b.bus.Subscribe(events.ReactionAdded, b.onReactionAdded)
	b.bus.Subscribe(events.ReactionRemoved, b.onReactionRemoved)

	// Channel events
	b.bus.Subscribe(events.ChannelCreated, b.onChannelCreated)
	b.bus.Subscribe(events.ChannelUpdated, b.onChannelUpdated)
	b.bus.Subscribe(events.ChannelDeleted, b.onChannelDeleted)

	// Server events
	b.bus.Subscribe(events.ServerCreated, b.onServerCreated)
	b.bus.Subscribe(events.ServerUpdated, b.onServerUpdated)
	b.bus.Subscribe(events.ServerDeleted, b.onServerDeleted)

	// Member events
	b.bus.Subscribe(events.MemberJoined, b.onMemberJoined)
	b.bus.Subscribe(events.MemberLeft, b.onMemberLeft)
	b.bus.Subscribe(events.MemberUpdated, b.onMemberUpdated)
	b.bus.Subscribe(events.MemberKicked, b.onMemberKicked)
	b.bus.Subscribe(events.MemberBanned, b.onMemberBanned)

	// User events
	b.bus.Subscribe(events.UserUpdated, b.onUserUpdated)
	b.bus.Subscribe(events.PresenceUpdate, b.onPresenceUpdate)

	// Typing events
	b.bus.Subscribe(events.TypingStarted, b.onTypingStarted)
}

// Message event handlers

type MessageEventData struct {
	Message   *models.Message `json:"message"`
	ChannelID uuid.UUID       `json:"channel_id"`
	ServerID  *uuid.UUID      `json:"server_id,omitempty"`
}

func (b *EventBridge) onMessageCreated(event events.Event) {
	data, ok := event.Data.(*MessageEventData)
	if !ok {
		return
	}

	wsData := b.messageToWS(data.Message)
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToChannel(data.ChannelID, &Event{
		Type: EventTypeMessageCreate,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onMessageUpdated(event events.Event) {
	data, ok := event.Data.(*MessageEventData)
	if !ok {
		return
	}

	wsData := b.messageToWS(data.Message)
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToChannel(data.ChannelID, &Event{
		Type: EventTypeMessageUpdate,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onMessageDeleted(event events.Event) {
	data, ok := event.Data.(*MessageDeleteData)
	if !ok {
		return
	}

	jsonData, _ := json.Marshal(data)
	b.hub.SendToChannel(data.ChannelID, &Event{
		Type: EventTypeMessageDelete,
		Data: json.RawMessage(jsonData),
	})
}

type MessageDeleteData struct {
	MessageID uuid.UUID  `json:"id"`
	ChannelID uuid.UUID  `json:"channel_id"`
	ServerID  *uuid.UUID `json:"guild_id,omitempty"`
}

func (b *EventBridge) onMessagePinned(event events.Event) {
	data, ok := event.Data.(*ChannelPinsData)
	if !ok {
		return
	}

	jsonData, _ := json.Marshal(data)
	b.hub.SendToChannel(data.ChannelID, &Event{
		Type: EventTypeChannelPinsUpdate,
		Data: json.RawMessage(jsonData),
	})
}

type ChannelPinsData struct {
	ChannelID          uuid.UUID `json:"channel_id"`
	ServerID           *uuid.UUID `json:"guild_id,omitempty"`
	LastPinTimestamp   string    `json:"last_pin_timestamp"`
}

// Reaction event handlers

type ReactionEventData struct {
	MessageID uuid.UUID  `json:"message_id"`
	ChannelID uuid.UUID  `json:"channel_id"`
	ServerID  *uuid.UUID `json:"guild_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	Emoji     string     `json:"emoji"`
}

func (b *EventBridge) onReactionAdded(event events.Event) {
	data, ok := event.Data.(*ReactionEventData)
	if !ok {
		return
	}

	jsonData, _ := json.Marshal(data)
	b.hub.SendToChannel(data.ChannelID, &Event{
		Type: EventTypeReactionAdd,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onReactionRemoved(event events.Event) {
	data, ok := event.Data.(*ReactionEventData)
	if !ok {
		return
	}

	jsonData, _ := json.Marshal(data)
	b.hub.SendToChannel(data.ChannelID, &Event{
		Type: EventTypeReactionRemove,
		Data: json.RawMessage(jsonData),
	})
}

// Channel event handlers

type ChannelEventData struct {
	Channel  *models.Channel `json:"channel"`
	ServerID uuid.UUID       `json:"server_id"`
}

func (b *EventBridge) onChannelCreated(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}

	wsData := b.channelToWS(data.Channel)
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToServer(data.ServerID, &Event{
		Type: EventTypeChannelCreate,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onChannelUpdated(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}

	wsData := b.channelToWS(data.Channel)
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToServer(data.ServerID, &Event{
		Type: EventTypeChannelUpdate,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onChannelDeleted(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}

	deleteData := map[string]interface{}{
		"id":       data.Channel.ID.String(),
		"guild_id": data.ServerID.String(),
	}
	jsonData, _ := json.Marshal(deleteData)

	b.hub.SendToServer(data.ServerID, &Event{
		Type: EventTypeChannelDelete,
		Data: json.RawMessage(jsonData),
	})
}

// Server event handlers

type ServerEventData struct {
	Server *models.Server `json:"server"`
}

func (b *EventBridge) onServerCreated(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}

	// Server create is sent to the user who created it
	wsData := b.serverToWS(data.Server)
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToUser(data.Server.OwnerID, &Event{
		Type: EventTypeServerCreate,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onServerUpdated(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}

	wsData := b.serverToWS(data.Server)
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToServer(data.Server.ID, &Event{
		Type: EventTypeServerUpdate,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onServerDeleted(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}

	deleteData := map[string]interface{}{
		"id": data.Server.ID.String(),
	}
	jsonData, _ := json.Marshal(deleteData)

	b.hub.SendToServer(data.Server.ID, &Event{
		Type: EventTypeServerDelete,
		Data: json.RawMessage(jsonData),
	})
}

// Member event handlers

type MemberEventData struct {
	ServerID uuid.UUID     `json:"server_id"`
	UserID   uuid.UUID     `json:"user_id"`
	User     *models.User  `json:"user,omitempty"`
	Member   *models.Member `json:"member,omitempty"`
	Reason   string        `json:"reason,omitempty"`
}

func (b *EventBridge) onMemberJoined(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}

	wsData := map[string]interface{}{
		"guild_id":  data.ServerID.String(),
		"joined_at": data.Member.JoinedAt.Format("2006-01-02T15:04:05.000Z"),
		"roles":     []string{},
	}
	
	if data.User != nil {
		wsData["user"] = b.userToWS(data.User)
	} else {
		wsData["user"] = map[string]interface{}{"id": data.UserID.String()}
	}
	
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToServer(data.ServerID, &Event{
		Type: EventTypeMemberJoin,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onMemberLeft(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}

	wsData := map[string]interface{}{
		"guild_id": data.ServerID.String(),
	}
	
	if data.User != nil {
		wsData["user"] = b.userToWS(data.User)
	} else {
		wsData["user"] = map[string]interface{}{"id": data.UserID.String()}
	}
	
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToServer(data.ServerID, &Event{
		Type: EventTypeMemberLeave,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onMemberUpdated(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}

	wsData := map[string]interface{}{
		"guild_id": data.ServerID.String(),
		"roles":    []string{}, // Would include actual roles
	}
	
	if data.User != nil {
		wsData["user"] = b.userToWS(data.User)
	} else {
		wsData["user"] = map[string]interface{}{"id": data.UserID.String()}
	}
	
	if data.Member != nil && data.Member.Nickname != nil {
		wsData["nick"] = *data.Member.Nickname
	}
	
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToServer(data.ServerID, &Event{
		Type: EventTypeMemberUpdate,
		Data: json.RawMessage(jsonData),
	})
}

func (b *EventBridge) onMemberKicked(event events.Event) {
	// Same as member left, but could include audit log entry
	b.onMemberLeft(event)
}

func (b *EventBridge) onMemberBanned(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}

	wsData := map[string]interface{}{
		"guild_id": data.ServerID.String(),
	}
	
	if data.User != nil {
		wsData["user"] = b.userToWS(data.User)
	} else {
		wsData["user"] = map[string]interface{}{"id": data.UserID.String()}
	}
	
	jsonData, _ := json.Marshal(wsData)

	b.hub.SendToServer(data.ServerID, &Event{
		Type: EventTypeBanAdd,
		Data: json.RawMessage(jsonData),
	})
}

// User event handlers

type UserEventData struct {
	User *models.User `json:"user"`
}

func (b *EventBridge) onUserUpdated(event events.Event) {
	data, ok := event.Data.(*UserEventData)
	if !ok {
		return
	}

	wsData := b.userToWS(data.User)
	jsonData, _ := json.Marshal(wsData)

	// Send to the user themselves
	b.hub.SendToUser(data.User.ID, &Event{
		Type: EventTypeUserUpdate,
		Data: json.RawMessage(jsonData),
	})
}

type PresenceEventData struct {
	UserID     uuid.UUID   `json:"user_id"`
	Status     string      `json:"status"`
	Activities []string    `json:"activities"`
	ServerIDs  []uuid.UUID `json:"server_ids"`
}

func (b *EventBridge) onPresenceUpdate(event events.Event) {
	data, ok := event.Data.(*PresenceEventData)
	if !ok {
		return
	}

	wsData := map[string]interface{}{
		"user": map[string]interface{}{
			"id": data.UserID.String(),
		},
		"status":        data.Status,
		"activities":    data.Activities,
		"client_status": map[string]string{},
	}
	jsonData, _ := json.Marshal(wsData)

	// Send to all servers the user is in
	for _, serverID := range data.ServerIDs {
		b.hub.SendToServer(serverID, &Event{
			Type: EventTypePresenceUpdate,
			Data: json.RawMessage(jsonData),
		})
	}
}

// Typing event handler

type TypingEventData struct {
	ChannelID uuid.UUID  `json:"channel_id"`
	ServerID  *uuid.UUID `json:"server_id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
}

func (b *EventBridge) onTypingStarted(event events.Event) {
	data, ok := event.Data.(*TypingEventData)
	if !ok {
		return
	}

	wsData := TypingStartData{
		ChannelID: data.ChannelID.String(),
		UserID:    data.UserID.String(),
		Timestamp: 0, // Would use actual timestamp
	}
	if data.ServerID != nil {
		wsData.GuildID = data.ServerID.String()
	}

	jsonData, _ := json.Marshal(wsData)
	b.hub.SendToChannel(data.ChannelID, &Event{
		Type: EventTypeTypingStart,
		Data: json.RawMessage(jsonData),
	})
}

// Conversion helpers

func (b *EventBridge) messageToWS(msg *models.Message) map[string]interface{} {
	if msg == nil {
		return nil
	}
	
	result := map[string]interface{}{
		"id":         msg.ID.String(),
		"channel_id": msg.ChannelID.String(),
		"content":    msg.Content,
		"timestamp":  msg.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		"pinned":     msg.Pinned,
		"type":       0,
		"tts":        msg.TTS,
		"mentions":   []interface{}{},
		"mention_roles": []string{},
		"attachments": []interface{}{},
		"embeds":     []interface{}{},
	}
	
	if msg.ServerID != nil {
		result["guild_id"] = msg.ServerID.String()
	}
	
	if msg.Author != nil {
		result["author"] = b.publicUserToWS(msg.Author)
	} else {
		result["author"] = map[string]interface{}{
			"id": msg.AuthorID.String(),
		}
	}
	
	if msg.EditedAt != nil && !msg.EditedAt.IsZero() {
		ts := msg.EditedAt.Format("2006-01-02T15:04:05.000Z")
		result["edited_timestamp"] = ts
	}
	
	return result
}

func (b *EventBridge) channelToWS(ch *models.Channel) map[string]interface{} {
	if ch == nil {
		return nil
	}
	
	result := map[string]interface{}{
		"id":       ch.ID.String(),
		"name":     ch.Name,
		"type":     ch.Type,
		"position": ch.Position,
	}
	
	if ch.ServerID != nil {
		result["guild_id"] = ch.ServerID.String()
	}
	
	if ch.Topic != "" {
		result["topic"] = ch.Topic
	}
	
	if ch.ParentID != nil {
		result["parent_id"] = ch.ParentID.String()
	}
	
	return result
}

func (b *EventBridge) serverToWS(srv *models.Server) map[string]interface{} {
	if srv == nil {
		return nil
	}
	
	result := map[string]interface{}{
		"id":       srv.ID.String(),
		"name":     srv.Name,
		"owner_id": srv.OwnerID.String(),
	}
	
	if srv.IconURL != nil {
		result["icon"] = *srv.IconURL
	}
	if srv.BannerURL != nil {
		result["banner"] = *srv.BannerURL
	}
	if srv.Description != nil {
		result["description"] = *srv.Description
	}
	
	return result
}

func (b *EventBridge) userToWS(user *models.User) map[string]interface{} {
	if user == nil {
		return nil
	}
	
	result := map[string]interface{}{
		"id":            user.ID.String(),
		"username":      user.Username,
		"discriminator": user.Discriminator,
		"bot":           false,
	}
	
	if user.AvatarURL != nil {
		result["avatar"] = *user.AvatarURL
	}
	
	return result
}

func (b *EventBridge) publicUserToWS(user *models.PublicUser) map[string]interface{} {
	if user == nil {
		return nil
	}
	
	result := map[string]interface{}{
		"id":            user.ID.String(),
		"username":      user.Username,
		"discriminator": user.Discriminator,
		"bot":           false,
	}
	
	if user.AvatarURL != nil {
		result["avatar"] = *user.AvatarURL
	}
	
	return result
}

// Additional event types for websocket
const (
	EventTypeChannelPinsUpdate = "CHANNEL_PINS_UPDATE"
	EventTypeBanAdd            = "GUILD_BAN_ADD"
	EventTypeUserUpdate        = "USER_UPDATE"
)
