package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"hearth/internal/events"
	"hearth/internal/models"
	"hearth/internal/services"
)

// DistributedEventBridge connects the domain event bus to the DistributedHub
// It broadcasts events to local clients AND publishes to Redis for other instances
type DistributedEventBridge struct {
	hub *DistributedHub
	bus *events.Bus
	ctx context.Context
}

// NewDistributedEventBridge creates a new distributed event bridge
func NewDistributedEventBridge(ctx context.Context, hub *DistributedHub, bus *events.Bus) *DistributedEventBridge {
	bridge := &DistributedEventBridge{
		hub: hub,
		bus: bus,
		ctx: ctx,
	}
	bridge.registerHandlers()
	return bridge
}

// sendToChannelDistributed marshals data and sends to a channel via Redis pub/sub
func (b *DistributedEventBridge) sendToChannelDistributed(channelID uuid.UUID, eventType string, data interface{}) {
	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()

	if err := b.hub.SendToChannelDistributed(ctx, channelID, eventType, data); err != nil {
		log.Printf("[DistributedEventBridge] failed to send %s to channel %s: %v", eventType, channelID, err)
	}
}

// sendToServerDistributed marshals data and sends to a server via Redis pub/sub
func (b *DistributedEventBridge) sendToServerDistributed(serverID uuid.UUID, eventType string, data interface{}) {
	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()

	if err := b.hub.SendToServerDistributed(ctx, serverID, eventType, data); err != nil {
		log.Printf("[DistributedEventBridge] failed to send %s to server %s: %v", eventType, serverID, err)
	}
}

// sendToUserDistributed marshals data and sends to a user via Redis pub/sub
func (b *DistributedEventBridge) sendToUserDistributed(userID uuid.UUID, eventType string, data interface{}) {
	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	defer cancel()

	if err := b.hub.SendToUserDistributed(ctx, userID, eventType, data); err != nil {
		log.Printf("[DistributedEventBridge] failed to send %s to user %s: %v", eventType, userID, err)
	}
}

// registerHandlers sets up event handlers for all domain events
func (b *DistributedEventBridge) registerHandlers() {
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

	// Interaction events
	b.bus.Subscribe(events.InteractionCreated, b.onInteractionCreated)
	b.bus.Subscribe(events.AutocompleteCreated, b.onAutocompleteCreated)
	b.bus.Subscribe(events.CommandExecuted, b.onCommandExecuted)
	b.bus.Subscribe(events.CommandResponseSent, b.onCommandResponseSent)
	b.bus.Subscribe(events.ModalSubmitted, b.onModalSubmitted)

	// DM events
	b.bus.Subscribe(events.DMRecipientAdded, b.onDMRecipientAdded)
	b.bus.Subscribe(events.DMRecipientRemoved, b.onDMRecipientRemoved)
	b.bus.Subscribe(events.GroupDMCreated, b.onGroupDMCreated)
	b.bus.Subscribe(events.GroupDMUpdated, b.onGroupDMUpdated)

	// Server boost events
	b.bus.Subscribe("server.boost_added", b.onServerBoostAdded)
	b.bus.Subscribe("server.boost_removed", b.onServerBoostRemoved)
}

// Message event handlers

func (b *DistributedEventBridge) onMessageCreated(event events.Event) {
	data, ok := event.Data.(*services.MessageCreatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onMessageCreated: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting MESSAGE_CREATE to channel %s (distributed)", data.ChannelID)
	b.sendToChannelDistributed(data.ChannelID, EventTypeMessageCreate, b.messageToWS(data.Message))
}

func (b *DistributedEventBridge) onMessageUpdated(event events.Event) {
	data, ok := event.Data.(*services.MessageUpdatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onMessageUpdated: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting MESSAGE_UPDATE to channel %s (distributed)", data.ChannelID)
	b.sendToChannelDistributed(data.ChannelID, EventTypeMessageUpdate, b.messageToWS(data.Message))
}

func (b *DistributedEventBridge) onMessageDeleted(event events.Event) {
	data, ok := event.Data.(*services.MessageDeletedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onMessageDeleted: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting MESSAGE_DELETE to channel %s (distributed)", data.ChannelID)
	b.sendToChannelDistributed(data.ChannelID, EventTypeMessageDelete, map[string]interface{}{
		"id":         data.MessageID.String(),
		"channel_id": data.ChannelID.String(),
	})
}

func (b *DistributedEventBridge) onMessagePinned(event events.Event) {
	data, ok := event.Data.(*services.MessagePinnedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onMessagePinned: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting CHANNEL_PINS_UPDATE to channel %s (distributed)", data.ChannelID)
	b.sendToChannelDistributed(data.ChannelID, EventTypeChannelPinsUpdate, map[string]interface{}{
		"channel_id": data.ChannelID.String(),
	})
}

// Reaction event handlers

func (b *DistributedEventBridge) onReactionAdded(event events.Event) {
	data, ok := event.Data.(*services.ReactionAddedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onReactionAdded: wrong type %T", event.Data)
		return
	}
	wsData := &ReactionEventData{
		MessageID: data.MessageID,
		ChannelID: data.ChannelID,
		UserID:    data.UserID,
		Emoji:     data.Emoji,
	}
	b.sendToChannelDistributed(data.ChannelID, EventTypeReactionAdd, wsData)
}

func (b *DistributedEventBridge) onReactionRemoved(event events.Event) {
	data, ok := event.Data.(*services.ReactionRemovedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onReactionRemoved: wrong type %T", event.Data)
		return
	}
	wsData := &ReactionEventData{
		MessageID: data.MessageID,
		ChannelID: data.ChannelID,
		UserID:    data.UserID,
		Emoji:     data.Emoji,
	}
	b.sendToChannelDistributed(data.ChannelID, EventTypeReactionRemove, wsData)
}

func (b *DistributedEventBridge) onReactionRemovedAll(event events.Event) {
	data, ok := event.Data.(*services.ReactionRemovedAllEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onReactionRemovedAll: wrong type %T", event.Data)
		return
	}
	b.sendToChannelDistributed(data.ChannelID, EventTypeReactionRemoveAll, map[string]interface{}{
		"message_id": data.MessageID.String(),
		"channel_id": data.ChannelID.String(),
	})
}

// Channel event handlers

func (b *DistributedEventBridge) onChannelCreated(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.ServerID, EventTypeChannelCreate, b.channelToWS(data.Channel))
}

func (b *DistributedEventBridge) onChannelUpdated(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.ServerID, EventTypeChannelUpdate, b.channelToWS(data.Channel))
}

func (b *DistributedEventBridge) onChannelDeleted(event events.Event) {
	data, ok := event.Data.(*ChannelEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.ServerID, EventTypeChannelDelete, map[string]interface{}{
		"id":       data.Channel.ID.String(),
		"guild_id": data.ServerID.String(),
	})
}

// Server event handlers

func (b *DistributedEventBridge) onServerCreated(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}
	// Server create is sent to the user who created it
	b.sendToUserDistributed(data.Server.OwnerID, EventTypeServerCreate, b.serverToWS(data.Server))
}

func (b *DistributedEventBridge) onServerUpdated(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.Server.ID, EventTypeServerUpdate, b.serverToWS(data.Server))
}

func (b *DistributedEventBridge) onServerDeleted(event events.Event) {
	data, ok := event.Data.(*ServerEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.Server.ID, EventTypeServerDelete, map[string]interface{}{
		"id": data.Server.ID.String(),
	})
}

// Member event handlers

func (b *DistributedEventBridge) onMemberJoined(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.ServerID, EventTypeMemberJoin, b.buildMemberData(data, true))
}

func (b *DistributedEventBridge) onMemberLeft(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.ServerID, EventTypeMemberLeave, b.buildMemberData(data, false))
}

func (b *DistributedEventBridge) onMemberUpdated(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}
	wsData := b.buildMemberData(data, false)
	wsData["roles"] = []string{} // Would include actual roles
	if data.Member != nil && data.Member.Nickname != nil {
		wsData["nick"] = *data.Member.Nickname
	}
	b.sendToServerDistributed(data.ServerID, EventTypeMemberUpdate, wsData)
}

func (b *DistributedEventBridge) onMemberKicked(event events.Event) {
	// Same as member left, but could include audit log entry
	b.onMemberLeft(event)
}

func (b *DistributedEventBridge) onMemberBanned(event events.Event) {
	data, ok := event.Data.(*MemberEventData)
	if !ok {
		return
	}
	b.sendToServerDistributed(data.ServerID, EventTypeBanAdd, b.buildMemberData(data, false))
}

// buildMemberData creates the common member event payload
func (b *DistributedEventBridge) buildMemberData(data *MemberEventData, includeJoinedAt bool) map[string]interface{} {
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

func (b *DistributedEventBridge) onUserUpdated(event events.Event) {
	data, ok := event.Data.(*UserEventData)
	if !ok {
		return
	}
	// Send to the user themselves
	b.sendToUserDistributed(data.User.ID, EventTypeUserUpdate, b.userToWS(data.User))
}

func (b *DistributedEventBridge) onPresenceUpdate(event events.Event) {
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
		b.sendToServerDistributed(serverID, EventTypePresenceUpdate, wsData)
	}
}

// Typing event handler

func (b *DistributedEventBridge) onTypingStarted(event events.Event) {
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
	b.sendToChannelDistributed(data.ChannelID, EventTypeTypingStart, wsData)
}

// Conversion helpers

func (b *DistributedEventBridge) messageToWS(msg *models.Message) map[string]interface{} {
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

func (b *DistributedEventBridge) channelToWS(ch *models.Channel) map[string]interface{} {
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

func (b *DistributedEventBridge) serverToWS(srv *models.Server) map[string]interface{} {
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

func (b *DistributedEventBridge) userToWS(user *models.User) map[string]interface{} {
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

func (b *DistributedEventBridge) publicUserToWS(user *models.PublicUser) map[string]interface{} {
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

// Interaction event handlers

func (b *DistributedEventBridge) onInteractionCreated(event events.Event) {
	data, ok := event.Data.(*services.InteractionCreatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onInteractionCreated: wrong type %T", event.Data)
		return
	}
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
	}
	b.sendToChannelDistributed(data.ChannelID, EventInteractionCreate, wsData)
}

func (b *DistributedEventBridge) onAutocompleteCreated(event events.Event) {
	data, ok := event.Data.(*services.InteractionCreatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onAutocompleteCreated: wrong type %T", event.Data)
		return
	}
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
	}
	b.sendToChannelDistributed(data.ChannelID, EventAutocomplete, wsData)
}

func (b *DistributedEventBridge) onCommandExecuted(event events.Event) {
	data, ok := event.Data.(*services.CommandExecutedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onCommandExecuted: wrong type %T", event.Data)
		return
	}
	wsData := map[string]interface{}{
		"command_id":    data.Execution.CommandID.String(),
		"app_id":        data.Execution.AppID.String(),
		"channel_id":    data.Execution.ChannelID.String(),
		"user_id":       data.Execution.UserID.String(),
		"status":        data.Status,
		"response_time": data.ResponseTime,
		"created_at":    data.Execution.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
	}
	if data.Execution.ServerID != nil {
		wsData["guild_id"] = data.Execution.ServerID.String()
	}
	// Broadcast to channel for logging/debugging
	b.sendToChannelDistributed(data.Execution.ChannelID, EventCommandExecute, wsData)
}

func (b *DistributedEventBridge) onCommandResponseSent(event events.Event) {
	data, ok := event.Data.(*services.CommandResponseSentEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onCommandResponseSent: wrong type %T", event.Data)
		return
	}
	wsData := map[string]interface{}{
		"token":      data.Token,
		"app_id":     data.AppID.String(),
		"channel_id": data.ChannelID.String(),
		"success":    data.Success,
	}
	if data.ServerID != nil {
		wsData["guild_id"] = data.ServerID.String()
	}
	b.sendToChannelDistributed(data.ChannelID, EventCommandResponse, wsData)
}

func (b *DistributedEventBridge) onModalSubmitted(event events.Event) {
	data, ok := event.Data.(*services.InteractionCreatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onModalSubmitted: wrong type %T", event.Data)
		return
	}
	wsData := map[string]interface{}{
		"id":        data.ID.String(),
		"type":      int(data.Type),
		"custom_id": data.Token,
		"user_id":   data.UserID.String(),
		"channel_id": data.ChannelID.String(),
	}
	if data.ServerID != nil {
		wsData["guild_id"] = data.ServerID.String()
	}
	b.sendToUserDistributed(data.UserID, EventModalSubmit, wsData)
}

// DM event handlers

func (b *DistributedEventBridge) onGroupDMCreated(event events.Event) {
	data, ok := event.Data.(*services.GroupDMCreatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onGroupDMCreated: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting GROUP_DM_CREATE for channel %s (distributed)", data.Channel.ID)
	b.sendToChannelDistributed(data.Channel.ID, EventTypeChannelCreate, b.channelToWS(data.Channel))
}

func (b *DistributedEventBridge) onGroupDMUpdated(event events.Event) {
	data, ok := event.Data.(*services.GroupDMUpdatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onGroupDMUpdated: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting GROUP_DM_UPDATE for channel %s (distributed)", data.Channel.ID)
	b.sendToChannelDistributed(data.Channel.ID, EventGroupDMUpdate, b.channelToWS(data.Channel))
}

func (b *DistributedEventBridge) onDMRecipientAdded(event events.Event) {
	data, ok := event.Data.(*services.DMRecipientEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onDMRecipientAdded: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting DM_RECIPIENT_ADD for channel %s, user %s (distributed)", data.ChannelID, data.UserID)

	wsData := map[string]interface{}{
		"channel_id": data.ChannelID.String(),
		"user_id":    data.UserID.String(),
	}

	if data.Channel != nil {
		wsData["channel"] = b.channelToWS(data.Channel)
	}

	b.sendToChannelDistributed(data.ChannelID, EventDMRecipientAdd, wsData)
}

func (b *DistributedEventBridge) onDMRecipientRemoved(event events.Event) {
	data, ok := event.Data.(*services.DMRecipientEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onDMRecipientRemoved: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting DM_RECIPIENT_REMOVE for channel %s, user %s (distributed)", data.ChannelID, data.UserID)

	wsData := map[string]interface{}{
		"channel_id": data.ChannelID.String(),
		"user_id":    data.UserID.String(),
	}

	b.sendToChannelDistributed(data.ChannelID, EventDMRecipientRemove, wsData)
}

// ServerBoostEvent represents a server boost change (for distributed bridge)
type ServerBoostDistributedData struct {
	ServerID    uuid.UUID `json:"server_id"`
	UserID      uuid.UUID `json:"user_id"`
	BoostCount  int       `json:"boost_count"`
	LevelBefore int       `json:"level_before"`
	LevelAfter  int       `json:"level_after"`
	Action      string    `json:"action"` // "added" or "removed"
}

func (b *DistributedEventBridge) onServerBoostAdded(event events.Event) {
	data, ok := event.Data.(*services.ServerBoostEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onServerBoostAdded: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting SERVER_BOOST_UPDATE to server %s (distributed)", data.ServerID)

	wsData := ServerBoostDistributedData{
		ServerID:    data.ServerID,
		UserID:      data.UserID,
		BoostCount:  data.BoostCount,
		LevelBefore: data.LevelBefore,
		LevelAfter:  data.LevelAfter,
		Action:      data.Action,
	}

	// Send to the server (distributed via Redis)
	b.sendToServerDistributed(data.ServerID, EventServerBoostUpdate, wsData)

	// If level changed, send level-specific events
	if data.LevelBefore != data.LevelAfter {
		if data.LevelAfter > data.LevelBefore {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelUp, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		} else {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelDown, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		}
	}
}

func (b *DistributedEventBridge) onServerBoostRemoved(event events.Event) {
	data, ok := event.Data.(*services.ServerBoostEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onServerBoostRemoved: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting SERVER_BOOST_UPDATE to server %s (distributed)", data.ServerID)

	wsData := ServerBoostDistributedData{
		ServerID:    data.ServerID,
		UserID:      data.UserID,
		BoostCount:  data.BoostCount,
		LevelBefore: data.LevelBefore,
		LevelAfter:  data.LevelAfter,
		Action:      data.Action,
	}

	// Send to the server (distributed via Redis)
	b.sendToServerDistributed(data.ServerID, EventServerBoostUpdate, wsData)

	// If level changed, send level-specific events
	if data.LevelBefore != data.LevelAfter {
		if data.LevelAfter > data.LevelBefore {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelUp, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		} else {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelDown, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		}
	}
}

// DM event handlers

func (b *DistributedEventBridge) onGroupDMCreated(event events.Event) {
	data, ok := event.Data.(*services.GroupDMCreatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onGroupDMCreated: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting GROUP_DM_CREATE for channel %s (distributed)", data.Channel.ID)
	b.sendToChannelDistributed(data.Channel.ID, EventTypeChannelCreate, b.channelToWS(data.Channel))
}

func (b *DistributedEventBridge) onGroupDMUpdated(event events.Event) {
	data, ok := event.Data.(*services.GroupDMUpdatedEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onGroupDMUpdated: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting GROUP_DM_UPDATE for channel %s (distributed)", data.Channel.ID)
	b.sendToChannelDistributed(data.Channel.ID, EventGroupDMUpdate, b.channelToWS(data.Channel))
}

func (b *DistributedEventBridge) onDMRecipientAdded(event events.Event) {
	data, ok := event.Data.(*services.DMRecipientEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onDMRecipientAdded: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting DM_RECIPIENT_ADD for channel %s, user %s (distributed)", data.ChannelID, data.UserID)

	wsData := map[string]interface{}{
		"channel_id": data.ChannelID.String(),
		"user_id":    data.UserID.String(),
	}

	if data.Channel != nil {
		wsData["channel"] = b.channelToWS(data.Channel)
	}

	b.sendToChannelDistributed(data.ChannelID, EventDMRecipientAdd, wsData)
}

func (b *DistributedEventBridge) onDMRecipientRemoved(event events.Event) {
	data, ok := event.Data.(*services.DMRecipientEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onDMRecipientRemoved: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting DM_RECIPIENT_REMOVE for channel %s, user %s (distributed)", data.ChannelID, data.UserID)

	wsData := map[string]interface{}{
		"channel_id": data.ChannelID.String(),
		"user_id":    data.UserID.String(),
	}

	b.sendToChannelDistributed(data.ChannelID, EventDMRecipientRemove, wsData)
}

// ServerBoostEvent represents a server boost change (for distributed bridge)
type ServerBoostDistributedData struct {
	ServerID    uuid.UUID `json:"server_id"`
	UserID      uuid.UUID `json:"user_id"`
	BoostCount  int       `json:"boost_count"`
	LevelBefore int       `json:"level_before"`
	LevelAfter  int       `json:"level_after"`
	Action      string    `json:"action"` // "added" or "removed"
}

func (b *DistributedEventBridge) onServerBoostAdded(event events.Event) {
	data, ok := event.Data.(*services.ServerBoostEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onServerBoostAdded: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting SERVER_BOOST_UPDATE to server %s (distributed)", data.ServerID)

	wsData := ServerBoostDistributedData{
		ServerID:    data.ServerID,
		UserID:      data.UserID,
		BoostCount:  data.BoostCount,
		LevelBefore: data.LevelBefore,
		LevelAfter:  data.LevelAfter,
		Action:      data.Action,
	}

	// Send to the server (distributed via Redis)
	b.sendToServerDistributed(data.ServerID, EventServerBoostUpdate, wsData)

	// If level changed, send level-specific events
	if data.LevelBefore != data.LevelAfter {
		if data.LevelAfter > data.LevelBefore {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelUp, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		} else {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelDown, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		}
	}
}

func (b *DistributedEventBridge) onServerBoostRemoved(event events.Event) {
	data, ok := event.Data.(*services.ServerBoostEvent)
	if !ok {
		log.Printf("[DistributedEventBridge] onServerBoostRemoved: wrong type %T", event.Data)
		return
	}
	log.Printf("[DistributedEventBridge] Broadcasting SERVER_BOOST_UPDATE to server %s (distributed)", data.ServerID)

	wsData := ServerBoostDistributedData{
		ServerID:    data.ServerID,
		UserID:      data.UserID,
		BoostCount:  data.BoostCount,
		LevelBefore: data.LevelBefore,
		LevelAfter:  data.LevelAfter,
		Action:      data.Action,
	}

	// Send to the server (distributed via Redis)
	b.sendToServerDistributed(data.ServerID, EventServerBoostUpdate, wsData)

	// If level changed, send level-specific events
	if data.LevelBefore != data.LevelAfter {
		if data.LevelAfter > data.LevelBefore {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelUp, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		} else {
			b.sendToServerDistributed(data.ServerID, EventServerBoostLevelDown, map[string]interface{}{
				"server_id":   data.ServerID.String(),
				"level":       data.LevelAfter,
				"boost_count": data.BoostCount,
			})
		}
	}
}

// Ensure all types needed exist or are imported
var _ = json.Marshal
