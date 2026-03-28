package websocket

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"

	"hearth/internal/events"
	"hearth/internal/models"
	"hearth/internal/services"
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

// sendToChannel marshals data and sends to a channel, logging errors
func (b *EventBridge) sendToChannel(channelID uuid.UUID, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[EventBridge] failed to marshal %s event: %v", eventType, err)
		return
	}
	b.hub.SendToChannel(channelID, &Event{
		Op:   OpDispatch,
		Type: eventType,
		Data: json.RawMessage(jsonData),
	})
}

// sendToServer marshals data and sends to a server, logging errors
func (b *EventBridge) sendToServer(serverID uuid.UUID, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[EventBridge] failed to marshal %s event: %v", eventType, err)
		return
	}
	b.hub.SendToServer(serverID, &Event{
		Op:   OpDispatch,
		Type: eventType,
		Data: json.RawMessage(jsonData),
	})
}

// sendToUser marshals data and sends to a user, logging errors
func (b *EventBridge) sendToUser(userID uuid.UUID, eventType string, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[EventBridge] failed to marshal %s event: %v", eventType, err)
		return
	}
	b.hub.SendToUser(userID, &Event{
		Op:   OpDispatch,
		Type: eventType,
		Data: json.RawMessage(jsonData),
	})
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
	b.bus.Subscribe(events.ReactionRemovedAll, b.onReactionRemovedAll)

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

	// Stream events
	b.bus.Subscribe("stream.started", b.onStreamStarted)
	b.bus.Subscribe("stream.ended", b.onStreamEnded)
	b.bus.Subscribe("stream.viewer_joined", b.onStreamViewerJoined)
	b.bus.Subscribe("stream.viewer_left", b.onStreamViewerLeft)

	// Interaction / slash command events
	b.bus.Subscribe(events.InteractionCreated, b.onInteractionCreated)
	b.bus.Subscribe(events.CommandExecuted, b.onCommandExecuted)
	b.bus.Subscribe(events.AutocompleteCreated, b.onAutocompleteCreated)
}

// Message event handlers

func (b *EventBridge) onMessageCreated(event events.Event) {
	data, ok := event.Data.(*services.MessageCreatedEvent)
	if !ok {
		log.Printf("[EventBridge] onMessageCreated: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting MESSAGE_CREATE to channel %s", data.ChannelID)
	b.sendToChannel(data.ChannelID, EventTypeMessageCreate, b.messageToWS(data.Message))
}

func (b *EventBridge) onMessageUpdated(event events.Event) {
	data, ok := event.Data.(*services.MessageUpdatedEvent)
	if !ok {
		log.Printf("[EventBridge] onMessageUpdated: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting MESSAGE_UPDATE to channel %s", data.ChannelID)
	b.sendToChannel(data.ChannelID, EventTypeMessageUpdate, b.messageToWS(data.Message))
}

func (b *EventBridge) onMessageDeleted(event events.Event) {
	data, ok := event.Data.(*services.MessageDeletedEvent)
	if !ok {
		log.Printf("[EventBridge] onMessageDeleted: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting MESSAGE_DELETE to channel %s", data.ChannelID)
	b.sendToChannel(data.ChannelID, EventTypeMessageDelete, map[string]interface{}{
		"id":         data.MessageID.String(),
		"channel_id": data.ChannelID.String(),
	})
}

func (b *EventBridge) onMessagePinned(event events.Event) {
	data, ok := event.Data.(*services.MessagePinnedEvent)
	if !ok {
		log.Printf("[EventBridge] onMessagePinned: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting CHANNEL_PINS_UPDATE to channel %s", data.ChannelID)
	b.sendToChannel(data.ChannelID, EventTypeChannelPinsUpdate, map[string]interface{}{
		"channel_id": data.ChannelID.String(),
	})
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
	data, ok := event.Data.(*services.ReactionAddedEvent)
	if !ok {
		log.Printf("[EventBridge] onReactionAdded: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting REACTION_ADD to channel %s", data.ChannelID)
	wsData := &ReactionEventData{
		MessageID: data.MessageID,
		ChannelID: data.ChannelID,
		UserID:    data.UserID,
		Emoji:     data.Emoji,
	}
	b.sendToChannel(data.ChannelID, EventTypeReactionAdd, wsData)
}

func (b *EventBridge) onReactionRemoved(event events.Event) {
	data, ok := event.Data.(*services.ReactionRemovedEvent)
	if !ok {
		log.Printf("[EventBridge] onReactionRemoved: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting REACTION_REMOVE to channel %s", data.ChannelID)
	wsData := &ReactionEventData{
		MessageID: data.MessageID,
		ChannelID: data.ChannelID,
		UserID:    data.UserID,
		Emoji:     data.Emoji,
	}
	b.sendToChannel(data.ChannelID, EventTypeReactionRemove, wsData)
}

func (b *EventBridge) onReactionRemovedAll(event events.Event) {
	data, ok := event.Data.(*services.ReactionRemovedAllEvent)
	if !ok {
		log.Printf("[EventBridge] onReactionRemovedAll: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting REACTION_REMOVE_ALL to channel %s", data.ChannelID)
	b.sendToChannel(data.ChannelID, EventTypeReactionRemoveAll, map[string]interface{}{
		"message_id": data.MessageID.String(),
		"channel_id": data.ChannelID.String(),
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
	b.sendToServer(data.ServerID, EventTypeChannelCreate, b.channelToWS(data.Channel))
}

func (b *EventBridge) onChannelUpdated(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}
	b.sendToServer(data.ServerID, EventTypeChannelUpdate, b.channelToWS(data.Channel))
}

func (b *EventBridge) onChannelDeleted(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}
	b.sendToServer(data.ServerID, EventTypeChannelDelete, map[string]interface{}{
		"id":       data.Channel.ID.String(),
		"guild_id": data.ServerID.String(),
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
	b.sendToUser(data.Server.OwnerID, EventTypeServerCreate, b.serverToWS(data.Server))
}

func (b *EventBridge) onServerUpdated(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}
	b.sendToServer(data.Server.ID, EventTypeServerUpdate, b.serverToWS(data.Server))
}

func (b *EventBridge) onServerDeleted(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}
	b.sendToServer(data.Server.ID, EventTypeServerDelete, map[string]interface{}{
		"id": data.Server.ID.String(),
	})
}

// Member event handlers

type MemberEventData struct {
	ServerID uuid.UUID      `json:"server_id"`
	UserID   uuid.UUID      `json:"user_id"`
	User     *models.User   `json:"user,omitempty"`
	Member   *models.Member `json:"member,omitempty"`
	Reason   string         `json:"reason,omitempty"`
}

func (b *EventBridge) onMemberJoined(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}
	b.sendToServer(data.ServerID, EventTypeMemberJoin, b.buildMemberData(data, true))
}

func (b *EventBridge) onMemberLeft(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}
	b.sendToServer(data.ServerID, EventTypeMemberLeave, b.buildMemberData(data, false))
}

func (b *EventBridge) onMemberUpdated(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}
	wsData := b.buildMemberData(data, false)
	wsData["roles"] = []string{} // Would include actual roles
	if data.Member != nil && data.Member.Nickname != nil {
		wsData["nick"] = *data.Member.Nickname
	}
	b.sendToServer(data.ServerID, EventTypeMemberUpdate, wsData)
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
	b.sendToServer(data.ServerID, EventTypeBanAdd, b.buildMemberData(data, false))
}

// buildMemberData creates the common member event payload
func (b *EventBridge) buildMemberData(data *MemberEventData, includeJoinedAt bool) map[string]interface{} {
	wsData := map[string]interface{}{
		"guild_id": data.ServerID.String(),
	}
	if data.User != nil {
		wsData["user"] = b.userToWS(data.User)
	} else {
		wsData["user"] = map[string]interface{}{"id": data.UserID.String()}
	}
	if includeJoinedAt && data.Member != nil {
		wsData["joined_at"] = data.Member.JoinedAt.Format("2006-01-02T15:04:05.000Z")
		wsData["roles"] = []string{}
	}
	return wsData
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
	// Send to the user themselves
	b.sendToUser(data.User.ID, EventTypeUserUpdate, b.userToWS(data.User))
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

	// Send to all servers the user is in
	for _, serverID := range data.ServerIDs {
		b.sendToServer(serverID, EventTypePresenceUpdate, wsData)
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
	b.sendToChannel(data.ChannelID, EventTypeTypingStart, wsData)
}

// Conversion helpers

func (b *EventBridge) messageToWS(msg *models.Message) map[string]interface{} {
	if msg == nil {
		return nil
	}

	result := map[string]interface{}{
		"id":            msg.ID.String(),
		"channel_id":    msg.ChannelID.String(),
		"content":       msg.Content,
		"timestamp":     msg.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
		"pinned":        msg.Pinned,
		"type":          0,
		"tts":           msg.TTS,
		"mentions":      []interface{}{},
		"mention_roles": []string{},
		"attachments":   []interface{}{},
		"embeds":        []interface{}{},
	}

	// Server ID is derived from channel, not stored on message
	// guild_id will be set by the caller if needed

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

// Stream event handlers

func (b *EventBridge) onStreamStarted(event events.Event) {
	data, ok := event.Data.(*services.StreamStartedEvent)
	if !ok {
		log.Printf("[EventBridge] onStreamStarted: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting STREAM_START to channel %s", data.ChannelID)
	wsData := map[string]interface{}{
		"stream":     data.Stream,
		"guild_id":   data.ServerID.String(),
		"channel_id": data.ChannelID.String(),
	}
	b.sendToChannel(data.ChannelID, EventStreamStart, wsData)
}

func (b *EventBridge) onStreamEnded(event events.Event) {
	data, ok := event.Data.(*services.StreamEndedEvent)
	if !ok {
		log.Printf("[EventBridge] onStreamEnded: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting STREAM_END to channel %s", data.ChannelID)
	wsData := map[string]interface{}{
		"stream_id":  data.StreamID.String(),
		"guild_id":   data.ServerID.String(),
		"channel_id": data.ChannelID.String(),
		"user_id":    data.UserID.String(),
	}
	b.sendToChannel(data.ChannelID, EventStreamEnd, wsData)
}

func (b *EventBridge) onStreamViewerJoined(event events.Event) {
	data, ok := event.Data.(*services.StreamViewerJoinedEvent)
	if !ok {
		log.Printf("[EventBridge] onStreamViewerJoined: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting STREAM_VIEWER_JOIN to channel %s", data.ChannelID)
	wsData := map[string]interface{}{
		"stream_id":    data.StreamID.String(),
		"user_id":      data.UserID.String(),
		"viewer_count": data.ViewerCount,
		"guild_id":     data.ServerID.String(),
		"channel_id":   data.ChannelID.String(),
	}
	b.sendToChannel(data.ChannelID, EventStreamViewerJoin, wsData)
}

func (b *EventBridge) onStreamViewerLeft(event events.Event) {
	data, ok := event.Data.(*services.StreamViewerLeftEvent)
	if !ok {
		log.Printf("[EventBridge] onStreamViewerLeft: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting STREAM_VIEWER_LEAVE to channel %s", data.ChannelID)
	wsData := map[string]interface{}{
		"stream_id":    data.StreamID.String(),
		"user_id":      data.UserID.String(),
		"viewer_count": data.ViewerCount,
		"guild_id":     data.ServerID.String(),
		"channel_id":   data.ChannelID.String(),
	}
	b.sendToChannel(data.ChannelID, EventStreamViewerLeave, wsData)
}

// Interaction event handlers

func (b *EventBridge) onInteractionCreated(event events.Event) {
	data, ok := event.Data.(*services.InteractionCreatedEvent)
	if !ok {
		log.Printf("[EventBridge] onInteractionCreated: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting INTERACTION_CREATE to channel %s", data.ChannelID)
	wsData := map[string]interface{}{
		"id":             data.ID.String(),
		"type":           data.Type,
		"application_id": data.AppID.String(),
		"user_id":        data.UserID.String(),
		"channel_id":     data.ChannelID.String(),
		"token":          data.Token,
	}
	if data.ServerID != nil {
		wsData["guild_id"] = data.ServerID.String()
		b.sendToServer(*data.ServerID, EventInteractionCreate, wsData)
	} else {
		b.sendToChannel(data.ChannelID, EventInteractionCreate, wsData)
	}
}

func (b *EventBridge) onCommandExecuted(event events.Event) {
	data, ok := event.Data.(*services.CommandExecutedEvent)
	if !ok {
		log.Printf("[EventBridge] onCommandExecuted: wrong type %T", event.Data)
		return
	}
	if data.Execution == nil {
		return
	}
	log.Printf("[EventBridge] Broadcasting COMMAND_EXECUTE to channel %s", data.Execution.ChannelID)
	wsData := map[string]interface{}{
		"command_id":     data.Execution.CommandID.String(),
		"application_id": data.Execution.AppID.String(),
		"user_id":        data.Execution.UserID.String(),
		"channel_id":     data.Execution.ChannelID.String(),
		"status":         data.Status,
		"response_time":  data.ResponseTime,
	}
	if data.Execution.ServerID != nil {
		wsData["guild_id"] = data.Execution.ServerID.String()
	}
	b.sendToChannel(data.Execution.ChannelID, EventCommandExecute, wsData)
}

func (b *EventBridge) onAutocompleteCreated(event events.Event) {
	data, ok := event.Data.(*services.InteractionCreatedEvent)
	if !ok {
		log.Printf("[EventBridge] onAutocompleteCreated: wrong type %T", event.Data)
		return
	}
	log.Printf("[EventBridge] Broadcasting AUTOCOMPLETE to user %s", data.UserID)
	wsData := map[string]interface{}{
		"id":             data.ID.String(),
		"type":           data.Type,
		"application_id": data.AppID.String(),
		"user_id":        data.UserID.String(),
		"channel_id":     data.ChannelID.String(),
		"token":          data.Token,
	}
	b.sendToUser(data.UserID, EventAutocomplete, wsData)
}

// Additional event types for websocket
const (
	EventTypeChannelPinsUpdate = "CHANNEL_PINS_UPDATE"
	EventTypeBanAdd            = "GUILD_BAN_ADD"
	EventTypeUserUpdate        = "USER_UPDATE"
)
